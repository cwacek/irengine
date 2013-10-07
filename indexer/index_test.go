package indexer

import "bytes"
import "testing"
import "io/ioutil"
import "encoding/json"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/logging"
import "github.com/cwacek/irengine/indexer/filters"
import log "github.com/cihub/seelog"

var  (

  TestDocuments = []filereader.Document{
    filters.LoadTestDocument("A02","Since I was a young boy; I played the silver ball."),
    filters.LoadTestDocument("A03","Since Ph.D's don't fly F-16 jets, but they might work for the CDC on the CDC-50 project"),
  }

  basicOutput = [][]byte{
    []byte("1. 'a' [1]: (A02, 1, {4})"),
    []byte("2. 'ball' [1]: (A02, 1, {11})"),
    []byte("3. 'boy' [1]: (A02, 1, {6})"),
    []byte("4. 'but' [1]: (A03, 1, {7})"),
    []byte("5. 'cdc' [2]: (A03, 2, {13,16})"),
    []byte("6. 'cdc50' [1]: (A03, 1, {16})"),
    []byte("7. 'dont' [1]: (A03, 1, {3})"),
    []byte("8. 'f16' [1]: (A03, 1, {5})"),
    []byte("9. 'fly' [1]: (A03, 1, {4})"),
    []byte("10. 'for' [1]: (A03, 1, {11})"),
    []byte("11. 'i' [2]: (A02, 2, {2,7})"),
    []byte("12. 'jets' [1]: (A03, 1, {6})"),
    []byte("13. 'might' [1]: (A03, 1, {9})"),
    []byte("14. 'on' [1]: (A03, 1, {14})"),
    []byte("15. 'phds' [1]: (A03, 1, {2})"),
    []byte("16. 'played' [1]: (A02, 1, {8})"),
    []byte("17. 'project' [1]: (A03, 1, {17})"),
    []byte("18. 'silver' [1]: (A02, 1, {10})"),
    []byte("19. 'since' [2]: (A02, 1, {1}) (A03, 1, {1})"),
    []byte("20. 'the' [3]: (A02, 1, {9}) (A03, 2, {12,15})"),
    []byte("21. 'they' [1]: (A03, 1, {8})"),
    []byte("22. 'was' [1]: (A02, 1, {3})"),
    []byte("23. 'work' [1]: (A03, 1, {10})"),
    []byte("24. 'young' [1]: (A02, 1, {5})"),
  }

)

func TestMarshalTerm(t *testing.T) {
  logging.SetupTestLogging()
  token := filereader.NewToken("james", filereader.TextToken)
  token.Position = 10
  token.DocId = "TestDocument"

  term := NewTermFromToken(token, NewPositionalPostingList)

  if bytes, err := json.Marshal(term); err != nil {
    log.Infof("Failed")
    t.Errorf("Error marshalling term %v: %v", term.String(), err)
  } else {
    log.Infof("Marshaled %v to %s", term.String(), bytes)
  }
}

func TestSingleTermIndex(t *testing.T) {
  logging.SetupTestLogging()

  defer func () {
    if x := recover(); x != nil {
      log.Criticalf("Error: %v",  x)
      log.Flush()
    }
  }()

  var index Indexer

  index = new(SingleTermIndex)
  if tempdir, err := ioutil.TempDir("", "index"); err == nil {

    if err2 := index.Init(tempdir, -1); err != nil {
      panic(err2)
    }

  } else {
    panic(err)
  }

  filterChain := filters.NewAcronymFilter("acronyms")
  filterChain = filterChain.Connect(filters.NewHyphenFilter("hyphens"), false)
  filterChain = filterChain.Connect(filters.NewLowerCaseFilter("lower"), false)
  index.AddFilter(filterChain)

  for _, document := range TestDocuments {
    log.Debugf("Inserting %s", document)
    index.Insert(document)
    log.Debugf("Finished inserting %s", document)
  }

  log.Debugf("Inserted all documents")

  output := new(bytes.Buffer)
  index.PrintLexicon(output)

  for i, expected := range basicOutput {
    if line, err := output.ReadBytes('\n'); err != nil {
      t.Errorf("Error reading lexicon output at line %d. Expected '%s'", i+1, expected)
      break
    } else {

      trimmed := bytes.TrimSpace(line)

      if ! bytes.Equal(trimmed, expected) {
        t.Errorf("Mismatched lexicon output at line %d: Expected '%s' (len: %d). Got '%s' (len: %d)",
        i+1, expected, len(expected), line, len(line))
      }
    }
  }

  index.Delete()
}

