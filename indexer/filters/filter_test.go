package filters

import "testing"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/logging"
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
  logging.SetupTestLogging()

	toLower := NewLowerCaseFilter("lowercase")

	input := NewFilterPipe("test")
	toLower.SetInput(input)
	postFilter := toLower.Output()

	toLower.Pull()

	done := make(chan int)

  go CompareFiltered(t, lower_output, postFilter, done, false)

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

func TestChainedFilters(t *testing.T) {
  logging.SetupTestLogging()

	input := NewFilterPipe("test")

  f_lower := NewLowerCaseFilter("lowercase")
  f_lower.SetInput(input)

  f_null := NewNullFilter("null1")
  f_null.Follow(f_lower, false)

	f_null.Pull()

  if head := f_null.Head(); head != f_lower {
    t.Errorf("Head of %v was %v. Expected %v", f_null, head, f_lower)
  }

	done := make(chan int)
  go CompareFiltered(t, lower_output, f_null.Output(), done, false)

	for _, tok := range test_input {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestChainedFilters Complete")
}

func TestMultipleOutputs(t *testing.T) {
  logging.SetupTestLogging()

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
  go CompareFiltered(t, lower_output, f_null.Output(), done, false)
  go CompareFiltered(t, lower_output, f_null2.Output(), done, false)

	for _, tok := range test_input {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
	<-done
  log.Infof("TestMultipleOutputs Complete")
}

