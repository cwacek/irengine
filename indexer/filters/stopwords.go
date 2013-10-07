package filters

import log "github.com/cihub/seelog"
import "bufio"
import "io"
import "github.com/cwacek/irengine/scanner/filereader"

type StopWordFilter struct {
    FilterPlumbing
    stopwords map[string]int
    removed int
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

func (f*StopWordFilter) Apply(tok *filereader.Token) []*filereader.Token {

    if _, ok := f.stopwords[tok.Text]; ok {
        f.removed += 1
        return nil
    }

    tok.Position -= f.removed

    return []*filereader.Token{tok}
}


func (f *StopWordFilter) NotifyDocComplete() {
    f.removed = 0
}
