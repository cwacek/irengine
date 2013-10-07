package indexer

import "fmt"
import "bufio"
import "os"
import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import "sync"

type SingleTermIndex struct {
	dataDir string

	filterChain filters.Filter
	stopWords   StopWordList

	lexicon Lexicon

	termCount     int
	documentCount int

  inserterRunning bool
  insertLock *sync.RWMutex
  shutdown chan bool
}

func (t *SingleTermIndex) String() string {
	return fmt.Sprintf("[SingleTermIndex words:%d docs:%d datadir:%s",
		t.termCount,
		t.documentCount,
		t.dataDir)
}

func (t *SingleTermIndex) Init(datadir string) error {
  t.dataDir = datadir

  if err := os.MkdirAll(datadir, 0775); err != nil {
    return err
  }

  // Initialize some stuff
	t.filterChain = nil
  t.stopWords = nil

  t.lexicon = NewTrieLexicon()

  t.termCount = 0
  t.documentCount = 0

  t.inserterRunning = false
  t.shutdown = make(chan bool)
  t.insertLock = new(sync.RWMutex)

  return nil
}

func (t *SingleTermIndex) SetStopWordList(sw StopWordList) {
  t.stopWords = sw
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
  log.Debugf("Added %s to filterchain. Now have %s", f, t.filterChain)

  t.filterChain.Pull()
}

func (t *SingleTermIndex) PrintLexicon(w *bufio.Writer) {

  var term *Term
  t.insertLock.RLock()

  for i, entry := range t.lexicon.Walk() {
    term = entry.(*Term)
    log.Debugf("Walking found term %v", term)
    _, err := w.WriteString(fmt.Sprintf("%d. '%s' [%d]: %s\n", i, term.Text(),
                  term.Tf(), term.PostingList()))

    if err != nil {
      panic(err)
    }

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

  if ! t.inserterRunning {
    go t.inserter()
  }

  t.insertLock.Lock()
  for token := range d.Tokens() {
    log.Debugf("Inserting %s into index input", token)
    input.Push(token)
  }

  log.Debugf("Finished inserting tokens from %s", d.Identifier())
}

// Read tokens from tokenStream and insert it into the 
// index
func (t *SingleTermIndex) inserter() {

  t.inserterRunning = true

  filterChainOut := t.filterChain.Output()
  log.Debugf("inserter process started listening on %v", filterChainOut)

  for {
    var token *filereader.Token
    select {
    case token = <- filterChainOut.Pipe:
      break
    case <- t.shutdown:
      log.Debugf("Got shutdown signal")
      return
    }

    if token.Type == filereader.NullToken {
      t.insertLock.Unlock()
      continue
    }

    log.Debugf("Read %s from the filter chain. Inserting into index", token)
    if term, ok := t.lexicon.FindTerm([]byte(token.Text)); ok {
      // We found the term
      log.Debugf("Found %s in the lexicon: %s", token.Text, term)

      term.Register(token)
    } else {

      term = NewTermFromToken(token , NewPositionalPostingList)

      // Insert the new term
      t.lexicon.Insert(term.(*Term))
    }
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
