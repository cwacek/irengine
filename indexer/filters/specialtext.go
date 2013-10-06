package filters

import log "github.com/cihub/seelog"
import "strings"
import "github.com/cwacek/irengine/scanner/filereader"

var (

  FileExtensions = map[string]bool {
    "aiff": true,
    "aif": true,
    "au": true,
    "avi": true,
    "bat": true,
    "bmp": true,
    "class": true,
    "java": true,
    "csk": true,
    "cvs": true,
    "dbf": true,
    "dif": true,
    "doc": true,
    "docx": true,
    "eps": true,
    "exe": true,
    "fm": true,
    "gif": true,
    "hqx": true,
    "htm": true,
    "html": true,
    "jpg": true,
    "mac": true,
    "map": true,
    "mdb": true,
    "mid": true,
    "midi": true,
    "mov": true,
    "qt": true,
    "mtb": true,
    "mtw": true,
    "pdf": true,
    "p": true,
    "t": true,
    "png": true,
    "ppt": true,
    "psd": true,
    "psp": true,
    "qxd": true,
    "ra": true,
    "sit": true,
    "tar": true,
    "tif": true,
    "txt": true,
    "wav": true,
    "xls": true,
    "xlsx": true,
    "zip": true,
  }
)

type FilenameFilter struct {
  FilterPlumbing
}

func NewFilenameFilter(id string) Filter {
  f := new(FilenameFilter)
  f.Id = id
  f.self = f
  return f
}

func (f *FilenameFilter) Apply(tok *filereader.Token) []*filereader.Token {
  results := make([]*filereader.Token, 0, 4)
  var newtok *filereader.Token

  parts := strings.Split(tok.Text, ".")

  if _, ok := FileExtensions[parts[len(parts)-1]]; ok {
    //This is a filename. return the file extension and the whole thing 
    // together
    log.Debugf("Found filename %s", tok)

    newtok = CloneWithText(tok, strings.Join(parts[:len(parts)-1], ""))
    results = append(results, newtok)
  }

  results = append(results, tok)
  return results
}
