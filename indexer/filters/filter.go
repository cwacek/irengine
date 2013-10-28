package filters

import "fmt"
import "strings"
import "errors"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

var SingleTermFilterSequence Filter

type FilterFactory interface {
	Instantiate() Filter
	Deserialize(string)
	Serialize() string
}

var filterFactory map[string]FilterFactory

func Register(name string, filter FilterFactory) {
	if filterFactory == nil {
		filterFactory = make(map[string]FilterFactory)
	}

	filterFactory[name] = filter
}

func Instantiate(filterName string) Filter {
	if generator, ok := filterFactory[filterName]; ok {
		return generator.Instantiate()
	} else {
		panic(errors.New("Unknown filter: " + filterName))
	}
}

func init() {
	if logger, err := log.LoggerFromConfigAsBytes(
		[]byte(`<seelog minlevel="info"></seelog>`)); err == nil {
		log.ReplaceLogger(logger)
	}

	f := NewDigitsFilter()
	f = f.Connect(NewDateFilter(), false)
	f = f.Connect(NewHyphenFilter(), false)
	f = f.Connect(NewSlashFilter(), false)
	f = f.Connect(NewAcronymFilter(), false)
	f = f.Connect(NewFilenameFilter(), false)
	f = f.Connect(NewLowerCaseFilter(), false)
	SingleTermFilterSequence = f
}

/*
A filter reads a sequence of tokens and returns a token representing anything
special parsed from the words. Filters can be chained together, in which case
actions are applied consecutively.
*/
type Filter interface {
	GetId() string

	//Get the filter at the head of the chain
	Head() Filter

	SetParent(Filter)

	Input() *FilterPipe
	SetInput(*FilterPipe)

	//Connect :f: after this filter. Returns the bottom of the chain (i.e. f)
	Connect(f Filter, force bool) Filter

	Follow(f Filter, force bool)
	Output() *FilterPipe

	Apply(*filereader.Token) []*filereader.Token
	Pull() *FilterPipe
	Terminate()

	// Notify the filter that the current document is complete.
	NotifyDocComplete()

	//Write the filter chain to string
	String() string
	// Write just this filter to string
	Serialize() string
}

/* A FilterPipe connects two filters together */
type FilterPipe struct {
	Id   string
	Pipe chan *filereader.Token
}

func NewFilterPipe(id string) *FilterPipe {
	fp := new(FilterPipe)
	fp.Id = id
	fp.Pipe = make(chan *filereader.Token, 10)
	return fp
}

func (f FilterPipe) String() string {
	return fmt.Sprintf("<%s:%p>", f.Id, f.Pipe)
}

func (f *FilterPipe) Push(t *filereader.Token) {
	f.Pipe <- t
}

func CloneWithText(t *filereader.Token, parts ...string) *filereader.Token {

	tok := t.Clone()
	tok.Text = strings.Join(parts, "")
	tok.Final = true // If you cloned it, you modified it, so it's final

	return tok
}
