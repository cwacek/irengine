package filters

import log "github.com/cihub/seelog"
import "regexp"
import "strings"
import "unicode"
import "github.com/cwacek/irengine/scanner/filereader"

var acronymRegex=regexp.MustCompile(`[A-Z][a-z]*(?:\.[A-Z][a-z]*)+`)

type AcronymFilter struct {
  FilterPlumbing
}

func NewAcronymFilter(id string) Filter {
  f := new(AcronymFilter)
  f.Id = id
  f.self = f
  return f
}

func (f *AcronymFilter) IsDestructive() bool {
  return true
}

func (f *AcronymFilter) Apply(tok *filereader.Token) (result []*filereader.Token) {

  result = make([]*filereader.Token, 1)
  var newtok *filereader.Token

  log.Debugf("Received '%s'.", tok)
  if acronymRegex.MatchString(tok.Text) {
    newtok = tok.Clone()
    newtok.Text = strings.Map(func(r rune) rune {
      switch r {
      case '.':
        return -1
      default:
        return unicode.ToLower(r)
      }
    }, tok.Text)

    newtok.Final = true
    result[0] = newtok

  } else {
    result[0] = tok
  }

  return
}
