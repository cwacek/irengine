package filters

import log "github.com/cihub/seelog"
import "testing"
import "fmt"
import "github.com/cwacek/irengine/scanner/filereader"
import "strings"


func LoadTestDocument(id, teststring string) filereader.Document {
  testDoc := filereader.NewTrecDocument(id)
  tokenizer := filereader.BadXMLTokenizer_FromReader(
                  strings.NewReader(teststring))

  for {
    token, ok := tokenizer.Next()

    if ok != nil {
      break
    }
    testDoc.Add(token)
  }

  return testDoc
}


func CompareFiltered(t *testing.T, expected []*filereader.Token,
                     actual *FilterPipe, signal chan int, verbose bool) {

		i := 0
		for filtered := range actual.Pipe {

			log.Debugf("Reading. Got %v", filtered)
      if i >= len(expected) {
        t.Error("Received addl unexpected token.")
        continue
      }

			if ! filtered.Eql(expected[i]) {
				t.Errorf("Expected %v, but got %v, at position %d",
                 expected[i], filtered, i)
			} else {
				log.Debugf("Filter success: %s = %s", expected[i], filtered)
			}

      if verbose {
        fmt.Printf("%s, ", filtered)
      }

			i += 1
		}

    fmt.Printf("\n")
		signal<- 1
	}
