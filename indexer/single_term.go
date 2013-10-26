package indexer

import "fmt"
import "io"
import "os"
import "bytes"
import "encoding/json"
import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import "sync"

type StoredDocInfo struct {
	Id        filereader.DocumentId
	HumanId   string
	TermCount int
}

func (info *StoredDocInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(*info)
}

func (info *StoredDocInfo) OrigIdent() string {
	return info.HumanId
}

func (info *StoredDocInfo) Identifier() filereader.DocumentId {
	return info.Id
}

func (info *StoredDocInfo) Len() int {
	return info.TermCount
}

type DocInfoMap map[filereader.DocumentId]*StoredDocInfo

func (m DocInfoMap) MarshalJSON() ([]byte, error) {
	var buf = new(bytes.Buffer)
	var bytes []byte
	var err error

	buf.WriteRune('[')

	for _, elem := range m {
		if bytes, err = json.Marshal(elem); err != nil {
			return nil, err
		}
		buf.Write(bytes)

		buf.WriteRune(',')
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteRune(']')
	return buf.Bytes(), nil
}

type SingleTermIndex struct {
	dataDir string

	filterChain filters.Filter

	lexicon Lexicon

	TermCount     int
	DocumentCount int

	DocumentMap DocInfoMap

	// utility vars
	inserterRunning bool
	insertLock      *sync.RWMutex
	shutdown        chan bool
}

func (t *SingleTermIndex) Save() {
	var persist PersistentLexicon

	switch t.lexicon.(type) {
	case PersistentLexicon:
		persist = t.lexicon.(PersistentLexicon)
		persist.SaveToDisk()

		if file, err := os.Create(persist.Location() + "docmap.txt"); err != nil {
			log.Critical("Error opening document map file: %v", err)
			panic(err)
		} else {
			if bytes, err := json.MarshalIndent(t.DocumentMap, "", "  "); err != nil {
				panic(err)
			} else {
				file.Write(bytes)
			}
			file.Close()
		}

	default:
		panic("Save to disk not supported")
		log.Critical("Save to disk not supported")
	}

}

func (t *SingleTermIndex) String() string {
	return fmt.Sprintf("{SingleTermIndex terms:%d docs:%d}",
		t.lexicon.Len(),
		t.DocumentCount)
}

func (t *SingleTermIndex) Init(lexicon Lexicon) error {

	// Initialize some stuff
	t.filterChain = nil

	t.lexicon = lexicon

	t.TermCount = 0
	t.DocumentCount = 0

	t.DocumentMap = make(DocInfoMap)

	t.inserterRunning = false
	t.shutdown = make(chan bool)
	t.insertLock = new(sync.RWMutex)

	return nil
}

func (t *SingleTermIndex) AddFilter(f filters.Filter) {

	if t.inserterRunning {
		panic("Tried to add a filter with inserter goroutine running")
	}

	if t.filterChain == nil {
		t.filterChain = f
	} else {
		t.filterChain = t.filterChain.Connect(f, false)
	}
	log.Infof("Added %s to filterchain. Now have %s", f, t.filterChain)

	t.filterChain.Pull()
}

func (t *SingleTermIndex) PrintLexicon(w io.Writer) {

	t.insertLock.RLock()
	t.lexicon.Print(w)
	switch t.lexicon.(type) {
	case PersistentLexicon:
		t.lexicon.(PersistentLexicon).PrintDiskStats(w)
	}
	t.insertLock.RUnlock()

	return
}

func (t *SingleTermIndex) Insert(d filereader.Document) {

	var input *filters.FilterPipe

	if t.filterChain == nil {
		// There's no existing filterchain, so just make it the
		// same as input
		t.filterChain = filters.NewNullFilter("null")
	}

	if input = t.filterChain.Head().Input(); input == nil {

		input = filters.NewFilterPipe("test")
		t.filterChain.Head().SetInput(input)
	}

	if !t.inserterRunning {
		go t.inserter()
	}

	//Print this if things go south
	defer func() {
		if x := recover(); x != nil {
			log.Warnf("Inserting tokens from %s", d.OrigIdent())
		}
	}()

	info := new(StoredDocInfo)
	info.HumanId = d.OrigIdent()
	info.Id = d.Identifier()
	t.DocumentMap[info.Id] = info

	t.insertLock.Lock()
	for token := range d.Tokens() {
		log.Debugf("Inserting %s into index input", token)
		input.Push(token)
	}

	t.lexicon.(PersistentLexicon).PrintDiskStats(os.Stdout)
	log.Infof("Inserted %d tokens from %s. Have %d documents with %d terms",
		d.Len(), d.OrigIdent(), t.Len(), t.lexicon.Len())
}

// Read tokens from tokenStream and insert it into the
// index
func (t *SingleTermIndex) inserter() {

	t.inserterRunning = true

	filterChainOut := t.filterChain.Output()
	log.Debugf("inserter process started listening on %v", filterChainOut)

	var termcounter = 0
	var info *StoredDocInfo

	for {
		var token *filereader.Token
		select {
		case token = <-filterChainOut.Pipe:
			break
		case <-t.shutdown:
			log.Debugf("Got shutdown signal")
			return
		}

		if token.Type == filereader.NullToken {
			t.DocumentCount += 1
			info = t.DocumentMap[token.DocId]
			info.TermCount = termcounter
			termcounter = 0
			t.insertLock.Unlock()
			continue
		}

		t.lexicon.InsertToken(token)
		termcounter++
	}

	log.Criticalf("Filter chain %s closed")
	t.inserterRunning = false
}

func (t *SingleTermIndex) Delete() {
	if err := os.RemoveAll(t.dataDir); err != nil {
		panic(err)
	}
	log.Debugf("sending shutdown signal")
	t.shutdown <- true
}

//Forces a block until insertion threads are done
func (t *SingleTermIndex) WaitInsert() {
	t.insertLock.RLock()
	t.insertLock.RUnlock()
}

func (t *SingleTermIndex) Len() int {
	return t.DocumentCount
}
