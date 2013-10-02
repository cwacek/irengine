package filters

import "strings"
import log "github.com/cihub/seelog"

type LowerCaseFilter struct {
	FilterPlumbing
}

func NewLowerCaseFilter(id string) Filter {
  f := new(LowerCaseFilter)
  f.Id = id
  f.self = f
  return f
}

type NullFilter struct {
  FilterPlumbing
}

func NewNullFilter(id string) Filter {
  f := new(NullFilter)
  f.Id = id
  f.self = f
  return f
}

func (f *LowerCaseFilter) IsDestructive() bool {
	return true
}

func (f *LowerCaseFilter) Apply() {
		for tok := range f.Input().Pipe {
      log.Debugf("Received '%s'", tok)
      converted := tok.Clone()
			converted.Text = strings.ToLower(tok.Text)
      log.Debugf("Translated to '%s'", converted)
			f.SendAll(converted)
		}

		f.Terminate()
  }

func (f *NullFilter) Apply() {
  for tok := range f.input.Pipe {
    log.Debugf("Read %v. Forwarding to ", tok)
    f.SendAll(tok)
  }

  f.Terminate()
}

