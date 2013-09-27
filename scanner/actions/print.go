package actions

import "path/filepath"
import "regexp"
import "fmt"
import "os"
import "flag"
import filereader "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

func PrintTokens(
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

      for doc := range fr.ReadAll() {
        fmt.Printf("Document %s (%d tokens)\n", doc.Identifier(), doc.Len())
      }

    }
  }
  return nil
}
