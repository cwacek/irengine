package filters

import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import porter "github.com/agonopol/go-stem/stemmer"

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

  defer func() {
    // If porter panics, use the current token
    if err := recover(); err != nil {
      log.Warnf("Porter Stemmer failed to stem: %s", tok.Text)
      result[0] = tok
    }
  }()

  term := []byte(tok.Text)
  stemmed := porter.Stem(term)

  /*stemmed := porter.StemWithoutLowerCasing(term)*/

  result[0] = CloneWithText(tok, string(stemmed))
  log.Debugf("Porter changed %s to %s", tok.Text, result[0].Text)

  return
}
