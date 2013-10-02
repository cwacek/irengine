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

func (f *LowerCaseFilter) apply() {
		for tok := range f.Input().Pipe {
      log.Debugf("Received '%s'", tok)
      converted := tok.Clone()
			converted.Text = strings.ToLower(tok.Text)
      log.Debugf("Translated to '%s'", converted)
			f.SendAll(converted)
		}

		f.Terminate()
  }

func (f *LowerCaseFilter) Pull() {
  log.Debugf("%v was Pulled", f)

  if !f.running {
    go f.apply()
    f.running = true
  }

  if f.parent != nil {
    f.parent.Pull()
  }
}

func (f *NullFilter) apply() {
  for tok := range f.input.Pipe {
    log.Debugf("Read %v. Forwarding to ", tok)
    f.SendAll(tok)
  }

  f.Terminate()
}

func (f *NullFilter) Pull() {
  log.Debugf("%v was Pulled. Will read from %v", f, f.input)

  if !f.running {
    go f.apply()
    f.running = true
  }

  if f.parent != nil {
    log.Debugf("Calling Pull on %v", f.parent)
    f.parent.Pull()
  }
}
