package filters

import "testing"
import "github.com/cwacek/irengine/scanner/filereader"
import "fmt"
import log "github.com/cihub/seelog"

var (
	test_input = []*filereader.Token{
		&filereader.Token{Text: "WELCOME", Type: filereader.TextToken},
		&filereader.Token{Text: "this", Type: filereader.TextToken},
		&filereader.Token{Text: "is", Type: filereader.TextToken},
		&filereader.Token{Text: "Jims", Type: filereader.TextToken},
		&filereader.Token{Text: "hOuSe", Type: filereader.TextToken},
		&filereader.Token{Text: "Not", Type: filereader.TextToken},
		&filereader.Token{Text: "Jims", Type: filereader.TextToken},
		&filereader.Token{Text: "HOUSE", Type: filereader.TextToken},
	}

	lower_output = []*filereader.Token{
		&filereader.Token{Text: "welcome", Type: filereader.TextToken},
		&filereader.Token{Text: "this", Type: filereader.TextToken},
		&filereader.Token{Text: "is", Type: filereader.TextToken},
		&filereader.Token{Text: "jims", Type: filereader.TextToken},
		&filereader.Token{Text: "house", Type: filereader.TextToken},
		&filereader.Token{Text: "not", Type: filereader.TextToken},
		&filereader.Token{Text: "jims", Type: filereader.TextToken},
		&filereader.Token{Text: "house", Type: filereader.TextToken},
	}
)

func TestLowerCase(t *testing.T) {

	toLower := NewLowerCaseFilter("lowercase")

	input := NewFilterPipe("test")
	toLower.SetInput(input)
	postFilter := toLower.Output()

	toLower.Pull()

	done := make(chan int)

  go CompareFiltered(t, lower_output, postFilter, done)

	for _, tok := range test_input {
		log.Debugf("Inserting %v into input", tok)
    input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestLowerCase Complete")
}

/* Check and see if each value pulled from actual is equivalent
to the ones in expected */
func CompareFiltered(t *testing.T, expected []*filereader.Token,
                     actual *FilterPipe, signal chan int) {

		i := 0
		for filtered := range actual.Pipe {

			log.Debugf("Reading. Got %v", filtered)

			if ! filtered.Eql(expected[i]) {
				t.Errorf("Expected %v, but got %v, at position %d",
                 expected[i], filtered, i)
			} else {
				log.Debugf("Filter success: %s = %s", expected[i], filtered)
			}

			i += 1
		}

		signal<- 1
	}

func TestChainedFilters(t *testing.T) {

	input := NewFilterPipe("test")

  f_lower := NewLowerCaseFilter("lowercase")
  f_lower.SetInput(input)

  f_null := NewNullFilter("null1")
  f_null.Follow(f_lower, false)

	f_null.Pull()

	done := make(chan int)
  go CompareFiltered(t, lower_output, f_null.Output(), done)

	for _, tok := range test_input {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestChainedFilters Complete")
}

func TestMultipleOutputs(t *testing.T) {
	input := NewFilterPipe("test")

  f_lower := NewLowerCaseFilter("lowercase")
  f_lower.SetInput(input)

  f_null := NewNullFilter("null1")
  f_null2 := NewNullFilter("null2")

  f_null.Follow(f_lower, false)
  f_null2.Follow(f_lower, false)

  f_null.Pull()
  f_null2.Pull()

	done := make(chan int)
  go CompareFiltered(t, lower_output, f_null.Output(), done)
  go CompareFiltered(t, lower_output, f_null2.Output(), done)

	for _, tok := range test_input {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
	<-done
  log.Infof("TestMultipleOutputs Complete")
}

func init() {
	var appConfig = `
  <seelog type="sync" minlevel='debug'>
  <outputs formatid="scanner">
    <filter levels="critical,error,warn,info">
      <console formatid="scanner" />
    </filter>
    <filter levels="debug">
      <console formatid="debug" />
    </filter>
  </outputs>
  <formats>
  <format id="scanner" format="test: [%LEV] %Msg%n" />
  <format id="debug" format="test: [%LEV] %Func :: %Msg%n" />
  </formats>
  </seelog>
`

	logger, err := log.LoggerFromConfigAsBytes([]byte(appConfig))

	if err != nil {
		fmt.Println(err)
		return
	}

	log.ReplaceLogger(logger)
}
