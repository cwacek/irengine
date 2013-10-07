package indexer

import "text/scanner"
import "io"

type StopWordMap struct {
  stopwords map[string]int
}

func (sw *StopWordMap) Contains(term string) bool {

  if _, ok := sw.stopwords[term]; ok {
    return true
  }

  return false
}

func StopWordMapFromText(r io.Reader) (StopWordList, error) {

  sw := new(StopWordMap)

  reader := new(scanner.Scanner).Init(r)
  reader.Mode = scanner.ScanStrings

  for {
    token := reader.Scan()

    if token == scanner.EOF {
      break
    }

    sw.stopwords[reader.TokenText()] = 0
  }

  return sw, nil
}
