package filters

import log "github.com/cihub/seelog"
import "bufio"
import "io"
import "os"
import "fmt"
import "github.com/cwacek/irengine/scanner/filereader"

func init() {
	Register("stopwords", &StopWordFilterFactory{})
}

type StopWordFilter struct {
	FilterPlumbing
	stopwords map[string]int
	removed   int
}

type StopWordFilterFactory struct {
	filename string
}

func (arg *StopWordFilterFactory) Instantiate() Filter {
	if file, err := os.Open(arg.filename); err != nil {
		panic("Cannot open " + arg.filename)
	} else {
		return NewStopWordFilterFromReader(file)
	}
}

func (arg *StopWordFilterFactory) Serialize() string {
	return fmt.Sprintf("%s", arg.filename)
}

func (arg *StopWordFilterFactory) Deserialize(input string) {
	arg.filename = input
}

func (f *StopWordFilter) Serialize() string {
	return fmt.Sprintf("%s{%d}", f.Id, len(f.stopwords))
}

func NewStopWordFilterFromReader(r io.Reader) Filter {

	sw := new(StopWordFilter)
	sw.stopwords = make(map[string]int)

	reader := bufio.NewScanner(r)
	reader.Split(bufio.ScanWords)

	for reader.Scan() {
		log.Debugf("Inserting %s into list", reader.Bytes())
		sw.stopwords[reader.Text()] = 0
	}

	sw.self = sw
	sw.Id = "stopwords"
	return sw
}

func (f *StopWordFilter) Apply(tok *filereader.Token) []*filereader.Token {

	if _, ok := f.stopwords[tok.Text]; ok {
		return nil
	}

	return []*filereader.Token{tok}
}

func (f *StopWordFilter) NotifyDocComplete() {
	f.removed = 0
}
