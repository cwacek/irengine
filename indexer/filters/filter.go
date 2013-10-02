package filters

import "fmt"
import "github.com/cwacek/irengine/scanner/filereader"

/* 
A filter reads a sequence of tokens and returns a token representing anything
special parsed from the words. Filters can be chained together, in which case
actions are applied consecutively.

*/
type Filter interface {
  IsDestructive() bool
  GetId() string

  SetParent(Filter)

  Input() *FilterPipe
  SetInput(*FilterPipe)

  Connect(f Filter, force bool)
  Follow(f Filter, force bool)
  Output() *FilterPipe

  Pull()
  Terminate()
}


/* A FilterPipe connects two filters together */
type FilterPipe struct {
  Id string
  Pipe chan *filereader.Token
}

func NewFilterPipe(id string) (*FilterPipe) {
  fp := new(FilterPipe)
  fp.Id = id
  fp.Pipe = make(chan *filereader.Token, 10)
  return fp
}

func (f FilterPipe) String() string {
  return fmt.Sprintf("<%s:%p>",f.Id, f.Pipe)
}

