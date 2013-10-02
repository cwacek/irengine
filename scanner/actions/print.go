package actions

import "path/filepath"
import "regexp"
import "fmt"
import "os"
import "flag"
import filereader "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

func readDoc(fr filereader.FileReader, sink chan string, done chan string) {
  for doc := range fr.ReadAll() {
    sink <- fmt.Sprintf("[%s] Document %s (%d tokens)\n", fr.Path(), doc.Identifier(), doc.Len())
  }
  done <- fr.Path()
  return
}

var (
workers chan string
output chan string
worker_count = 0
)

func PrintTokens(docroot string) {
  output = make(chan string)
  workers = make(chan string)
  filepath.Walk(docroot, read_file)

  for {
    select {
    case <- output:
      fmt.Print("")
    case file := <- workers:
      worker_count -= 1
      log.Errorf("Worker for %s done. Waiting for %d workers.", file, worker_count)
      if worker_count <= 0 {
        return
      }
    }
  }

}

func read_file(
	path string, info os.FileInfo, err error) error {

  if info.Mode().IsRegular() {
    file := filepath.Base(path)

    log.Debugf("Trying file %s", file)

    pattern := flag.Lookup("doc.pattern")
    matched, err := regexp.MatchString(pattern.Value.String(), file);
    log.Debugf("File match: %v, error: %v", matched, err)
    if matched && err == nil {
      fmt.Printf("In file %s: ", path)

      fr := new(filereader.TrecFileReader)
      fr.Init(path)

      go readDoc(fr, output, workers)
      worker_count += 1
      log.Errorf("Now have %d workers", worker_count)
      /*for doc := range fr.ReadAll() {*/
        /*fmt.Printf("[%s] Document %s (%d tokens)\n", fr.Path(), doc.Identifier(), doc.Len())*/
      /*}*/

    }
  }
  return nil
}
