package filters

import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

type FilterPlumbing struct {
  Id string

  parent Filter
  input *FilterPipe
  output []*FilterPipe
  self Filter

  running bool
}

func (fc *FilterPlumbing) Input() (*FilterPipe) {
  return fc.input
}

func (fc *FilterPlumbing) SetInput(input_fp *FilterPipe) {
  fc.input = input_fp
}

func (fc *FilterPlumbing) Output() (*FilterPipe) {
  if len(fc.output) > 0 {
    panic("Tried to grab output of something with connected filters")
  }

  newconn := NewFilterPipe("out:" +fc.GetId())
  fc.output = append(fc.output, newconn)
  return newconn
}

func (fc *FilterPlumbing) SetParent(f Filter) {
  fc.parent = f
}

func (fc *FilterPlumbing) IsDestructive() bool {
  return false
}

func (fc *FilterPlumbing) GetId() string {
  return fc.Id
}

func (fc *FilterPlumbing) Follow(f Filter, force bool) {
  f.Connect(fc.self, force)
}

func (fc *FilterPlumbing) Connect(f Filter, force bool) {
  log.Debugf("Connecting %v after %v", f, fc)

  if f.Input() != nil && ! force {
      panic("Asked to connect to a filter with an existing input, without forcing")

  }

  newconn := NewFilterPipe(f.GetId() + ":connect:" + fc.Id)
  f.SetInput(newconn)
  f.SetParent(fc.self)
  fc.output = append(fc.output, newconn)
}

func (fc *FilterPlumbing) SendAll(tok *filereader.Token) {
  log.Debugf("Sending '%s' to %d output pipes: %v", tok, len(fc.output), fc.output)
  for _, out := range fc.output {
    out.Pipe <- tok
  }
}

func (fc *FilterPlumbing) Terminate() {
  for _, out := range fc.output {
    close(out.Pipe)
  }
}

func (fc *FilterPlumbing) Pull() {
  if !fc.running {
    go fc.self.Apply()
    fc.running = true
  }

  if fc.parent != nil {
    fc.parent.Pull()
  }
}
