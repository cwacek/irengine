package filters

import "github.com/cwacek/irengine/scanner/filereader"
import "strings"

var sw_input = `The quick brown fox jumped over a lazy dog`
var sw_stopwords = `the quick a`

var sw_expected = []*filereader.Token{
  &filereader.Token{Text: "brown", Type: filereader.TextToken},
  &filereader.Token{Text: "fox", Type: filereader.TextToken},
  &filereader.Token{Text: "jumped", Type: filereader.TextToken},
  &filereader.Token{Text: "over", Type: filereader.TextToken},
  &filereader.Token{Text: "lazy", Type: filereader.TextToken},
  &filereader.Token{Text: "dog", Type: filereader.TextToken},
}

func MakeStopWordFilter(id string) Filter {
    f := NewStopWordFilterFromReader(strings.NewReader(sw_stopwords))
    f.Follow(NewLowerCaseFilter("lowercase"),false)
    return f
}

func init() {

  AddTestCase("stopwords",TestCase{
    MakeStopWordFilter,
    sw_input,
    sw_expected,
  })

}

