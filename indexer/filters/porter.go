package filters

import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import porter "github.com/reiver/go-porterstemmer"


type PorterFilter struct {
  FilterPlumbing
}

func NewPorterFilter(id string) Filter {
  f := new(PorterFilter)
  f.Id = id
  f.self = f
  return f
}

func (f *PorterFilter) Apply(tok *filereader.Token) (result []*filereader.Token) {
  result = make([]*filereader.Token, 1)

  result[0] = CloneWithText(tok, porter.StemString(tok.Text))
  log.Debugf("Porter changed %s to %s", tok.Text, result[0].Text)

  return
}
