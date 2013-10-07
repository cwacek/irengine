package indexer

import "bufio"
/*import "testing"*/
import "os"
import "io/ioutil"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/logging"
import "github.com/cwacek/irengine/indexer/filters"
import log "github.com/cihub/seelog"

var  (

  TestDocuments = []filereader.Document{
    filters.LoadTestDocument("A02","Since I was a young boy; I played the silver ball."),
    filters.LoadTestDocument("A03","Since Ph.D's don't fly F-16 jets, but they might work for the CDC on the CDC-50 project"),
  }
)

func ExampleSingleTermIndex() {
/*func TestSingleTermIndex(t *testing.T) {*/
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

    if err2 := index.Init(tempdir); err != nil {
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

  writer := bufio.NewWriter(os.Stdout)
  index.PrintLexicon(writer)
  writer.Flush()

  index.Delete()

  // Output:
  // 1. 'a' [1]: (A02, 1, {4})
  // 2. 'ball' [1]: (A02, 1, {11})
  // 3. 'boy' [1]: (A02, 1, {6})
  // 5. 'but' [1]: (A03, 1, {7})
  // 6. 'cdc' [1]: (A03, 2, {13,16})
  // 7. 'dont' [1]: (A03, 1, {3})
  // 8. 'f16' [1]: (A03, 1, {4})
  // 9. 'fly' [1]: (A03, 1, {4})
  // 10. 'for' [1]: (A03, 1, {11})
  // 11. 'i' [2]: (A02, 2, {2,7})
  // 12. 'jets' [1]: (A03, 1, {6})
  // 12. 'might' [1]: (A03, 1, {9})
  // 13. 'on' [1]: (A03, 1, {14})
  // 14. 'phds' [1]: (A03, 1, {2})
  // 15. 'played' [1]: (A02, 1, {8})
  // 16. 'project' [1]: (A03, 1, {17})
  // 17. 'silver' [1]: (A02, 1, {10})
  // 18. 'since' [2]: (A02, 1, {1}) (A03, 1, {1})
  // 19. 'the' [3]: (A02, 1, {9}) (A03, 2, {12,15})
  // 20. 'they' [1]: (A03, 1, {8})
  // 21. 'was' [1]: (A02, 1, {3})
  // 22. 'work' [1]: (A03, 1, {10})
  // 23. 'young' [1]: (A02, 1, {5})
}
