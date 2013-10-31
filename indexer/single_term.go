package indexer

import "fmt"
import "io"
import "math"
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
	MaxTf     int
	// Store the term frequency for every term
	// in this document. This is heinous, but
	// Cosine doesn't work without it, and no
	// one can adequately explain how that makes
	// any fucking sense. You have to iterate
	// over all the terms *in a document*........
	TermTfIdf map[string]float64
}

func (info *StoredDocInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(*info)
}

func (info *StoredDocInfo) Clone() (new_info *StoredDocInfo) {
	new_info = new(StoredDocInfo)
	new_info.HumanId = info.HumanId
	new_info.TermCount = info.TermCount
	new_info.Id = info.Id
	new_info.MaxTf = info.MaxTf
	new_info.TermTfIdf = info.TermTfIdf

	return
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

func (m DocInfoMap) UnmarshalJSON(input []byte) (e error) {

	info_array := make([]StoredDocInfo, 0)
	if e = json.Unmarshal(input, &info_array); e != nil {
		return e
	}

	for _, info := range info_array {
		m[info.Id] = (&info).Clone()
	}

	return nil
}

type SingleTermIndex struct {
	dataDir string

	filterChain filters.Filter

	lexicon Lexicon

	DocumentCount int

	DocumentMap DocInfoMap

	// utility vars
	inserterRunning bool
	insertLock      *sync.RWMutex
	shutdown        chan bool
}

func (t *SingleTermIndex) TermCount() int {
	return t.lexicon.Len()
}

func (t *SingleTermIndex) Retrieve(text string) (LexiconTerm, bool) {
	return t.lexicon.FindTerm([]byte(text))
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

		if file, err := os.Create(persist.Location() + "filters.mdt"); err != nil {
			log.Criticalf("Error opening filter file: %v", err)
			panic(err)
		} else {

			var filterFactory filters.FilterFactory
			var e error
			var out string

			log.Warnf("filterchain Ids: %v", t.filterChain.Ids())
			for _, filter := range t.filterChain.Ids() {
				log.Infof("Writing '%s' to filter metadata", filter)
				if filterFactory, e = filters.GetFactory(filter); e == nil {
					out = filter + " " + filterFactory.Serialize()
					fmt.Fprintln(file, out)
				} else {
					log.Warnf("Couldn't save %s because don't know how.", filter)
				}
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
	log.Debugf("Added %s to filterchain. Now have %s", f.Serialize(), t.filterChain)

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

/* Connect the filter chain to the input and output channels and translate between them. */
func (t *SingleTermIndex) FilterTokens(input, output chan *filereader.Token) {

	if t.filterChain == nil {
		// There's no existing filterchain, so just make it the
		// same as input
		t.filterChain = filters.NewNullFilter()
	}

	log.Debugf("Filtering tokens")
	t.filterChain.Head().SetInput(&filters.FilterPipe{"input", input})
	t.filterChain.Pull()

	filterChainOut := t.filterChain.Output()

	var token *filereader.Token
	var ok bool
	for {
		log.Tracef("Reading from filter chain")

		token, ok = <-filterChainOut.Pipe
		if !ok {
			return
		}

		log.Tracef("Read %v", token)

		output <- token
	}

	return
}

func (t *SingleTermIndex) Insert(d filereader.Document) {

	var input *filters.FilterPipe

	if t.filterChain == nil {
		// There's no existing filterchain, so just make it the
		// same as input
		t.filterChain = filters.NewNullFilter()
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
	info.TermTfIdf = make(map[string]float64)
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
	var term LexiconTerm

	for {
		var token *filereader.Token
		select {
		case token = <-filterChainOut.Pipe:
			break
		case <-t.shutdown:
			log.Debugf("Got shutdown signal")
			return
		}

		info = t.DocumentMap[token.DocId]
		if token.Type == filereader.NullToken {
			t.DocumentCount += 1
			info.TermCount = termcounter
			termcounter = 0
			t.insertLock.Unlock()
			continue
		}

		term = t.lexicon.InsertToken(token)

		/* Update document-indexed statistics */

		// Add 1 to the document count to make sure we count this one
		weight := Tf_d(term, info.Id) * Idf(term, t.DocumentCount+1)

		log.Debugf("Setting weight for %s in %s to %0.4f * %0.4f = %0.4f",
			token.Text, info.HumanId, Tf_d(term, info.Id), Idf(term, t.DocumentCount), weight)
		if math.IsInf(weight, 0) {
			panic("TO INFINITY AND BEYOND!")
		}

		info.TermTfIdf[term.Text()] = weight

		if term.Tf() > info.MaxTf {
			info.MaxTf = term.Tf()
		}
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

func Tf_d(t LexiconTerm, d filereader.DocumentId) float64 {
	pl_entry, ok := t.PostingList().GetEntry(d)
	if !ok {
		return 0.0
	}

	return float64(pl_entry.Frequency())
}

func Df(t LexiconTerm) int {
	return t.PostingList().Len()
}

func Idf(t LexiconTerm, totalDocCount int) float64 {
	plLen := float64(Df(t))
	return math.Log10((float64(totalDocCount) - plLen + 0.5) / (plLen + 0.5))
}
