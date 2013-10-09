package filters

import "github.com/cwacek/irengine/scanner/filereader"
import "strings"
import log "github.com/cihub/seelog"

type FilterPlumbing struct {
  Id string

  parent Filter
  input *FilterPipe
  output []*FilterPipe
  self Filter

  running bool
  ignoresFinal bool
}

func (fc *FilterPlumbing) Head() Filter {
  log.Debugf("Looking for head in %v, which has parent %v", fc.self, fc.parent)
  if fc.parent != nil {
    return fc.parent.Head()
  } else {
    return fc.self
  }
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

func (fc *FilterPlumbing) Connect(f Filter, force bool) Filter{
  log.Debugf("Connecting %v after %v", f, fc)

  if f.Input() != nil && ! force {
      panic("Asked to connect to a filter with an existing input, without forcing")

  }

  newconn := NewFilterPipe(f.GetId() + ":connect:" + fc.Id)
  f.SetInput(newconn)
  f.SetParent(fc.self)
  fc.output = append(fc.output, newconn)
  return f
}

func  (fc *FilterPlumbing) String() string {
  parts := make([]string, 0)

  if fc.parent != nil {
    parts = append(parts,fc.parent.String())
  }

  parts = append(parts, fc.self.Serialize())

  return strings.Join(parts, " -> ")
}

func (fc *FilterPlumbing) Serialize() string {
  return fc.Id
}

func (fc *FilterPlumbing) Send(tok *filereader.Token) {
  log.Debugf("Sending '%s' to %d output pipes: %v", tok, len(fc.output), fc.output)
  for _, out := range fc.output {
    out.Pipe <- tok
  }
}

func (fc *FilterPlumbing) SendAll(tokens []*filereader.Token) {

  log.Debugf("%s Sending '%s' to %d output pipes: %v",
             fc.Id, tokens, len(fc.output), fc.output)

  for _, out := range fc.output {
    for _, tok := range tokens {
      out.Pipe <- tok
    }
  }
}

func (fc *FilterPlumbing) Terminate() {
  for _, out := range fc.output {
    log.Debugf("Filter %s terminating", fc)
    close(out.Pipe)
  }
}

func (fc *FilterPlumbing) apply() {
  log.Debugf("Applying %v. Reading %v", fc, fc.Input())
  for tok := range fc.Input().Pipe {
    log.Debugf("%s received %s", fc.Id, tok)

    switch {
    case tok.Type == filereader.SymbolToken:
      // Don't pass it along

    case tok.Type == filereader.NullToken:
      fc.self.NotifyDocComplete()
      fc.Send(tok)

    case tok.Final && fc.ignoresFinal == false:
      log.Tracef("Passing Final token %s along", tok)
      fc.Send(tok)

    default:
      fc.SendAll(fc.self.Apply(tok))
    }

  }

  fc.Terminate()

}

func (fc *FilterPlumbing) NotifyDocComplete() {

}

func (fc *FilterPlumbing) Pull() *FilterPipe {
  log.Debugf("Pulled %v. Have parent %v and input %v", fc, fc.parent, fc.input)
  if !fc.running {
    go fc.apply()
    fc.running = true
  }

  if fc.parent != nil {
    return fc.parent.Pull()
  } else {

    var input *FilterPipe

    if fc.input == nil {
      input = NewFilterPipe("input")
      fc.SetInput(input)
    }
    return input
  }
}
