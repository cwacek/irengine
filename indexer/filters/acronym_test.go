package filters

import "testing"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

var (
  acro_input = `Ph.D. Ph.D Phd PhD U.S.A USA M.S. M.S MS `
  hyphens_input = `CDC-50 F-16 1-hour`

	acro_output = []*filereader.Token{
    &filereader.Token{Text: "phd", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "phd", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "Phd", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "PhD", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "usa", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "USA", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "ms", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "ms", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "MS", Type: filereader.TextToken, Final: true},
	}


	hyphens_out = []*filereader.Token{
    &filereader.Token{Text: "CDC50", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "CDC", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "F16", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "1hour", Type: filereader.TextToken, Final: true},
    &filereader.Token{Text: "hour", Type: filereader.TextToken, Final: true},
	}

  chained_in string
  chained_out []*filereader.Token
)

func init() {

  chained_in = acro_input + hyphens_input
  chained_out = append(acro_output, hyphens_out...)
}

func TestChained(t *testing.T) {
  SetupTestLogging()

	filter1:= NewAcronymFilter("acronyms")
	filter2 := NewHyphenFilter("hyphens")
  filter2.Follow(filter1, false)

	postFilter := filter2.Output()

	input := NewFilterPipe("test")
  log.Debugf("Created input pipe %v", input)
	filter1.SetInput(input)

  filter2.Pull()

	done := make(chan int)

  go CompareFiltered(t, chained_out, postFilter, done)

	for  tok := range LoadTestDocument(chained_in).Tokens() {
		log.Debugf("Inserting %v into input", tok)
    input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestChained Complete")
}

func TestAcronyms(t *testing.T) {
  SetupTestLogging()

	input := NewFilterPipe("test")
  log.Debugf("Created input pipe %v", input)

	filter := NewAcronymFilter("acronyms")
  log.Debugf("Created filter %v", filter)
  log.Flush()
	filter.SetInput(input)
	postFilter := filter.Output()

	filter.Pull()

	done := make(chan int)

  go CompareFiltered(t, acro_output, postFilter, done)

  testDoc := LoadTestDocument(acro_input)
	for  tok := range testDoc.Tokens(){
		log.Debugf("Inserting %v into input", tok)
    input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestAcronyms Complete")
}

func TestHyphens(t *testing.T) {
  SetupTestLogging()

  input := NewFilterPipe("test")

  filter := NewHyphenFilter("hyphens")
  filter.SetInput(input)

  postFilter := filter.Output()

  filter.Pull()

  done := make(chan int)

  go CompareFiltered(t, hyphens_out, postFilter, done)

  testDoc := LoadTestDocument(hyphens_input)
	for  tok := range testDoc.Tokens(){
		log.Debugf("Inserting %v into input", tok)
    input.Push(tok)
	}

	close(input.Pipe)
	<-done
  log.Infof("TestHyphens Complete")
}
