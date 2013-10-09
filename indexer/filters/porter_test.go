package filters

import "github.com/cwacek/irengine/scanner/filereader"

var expected = []*filereader.Token {
  &filereader.Token{Text: "aaron", Type: filereader.TextToken},
  &filereader.Token{Text: "abaissiez", Type: filereader.TextToken},
  &filereader.Token{Text: "abandon", Type: filereader.TextToken},
  &filereader.Token{Text: "abandon", Type: filereader.TextToken},
  &filereader.Token{Text: "abas", Type: filereader.TextToken},
  &filereader.Token{Text: "abash", Type: filereader.TextToken},
  &filereader.Token{Text: "abat", Type: filereader.TextToken},
  &filereader.Token{Text: "abat", Type: filereader.TextToken},
  &filereader.Token{Text: "abat", Type: filereader.TextToken},
  &filereader.Token{Text: "abat", Type: filereader.TextToken},
  &filereader.Token{Text: "abat", Type: filereader.TextToken},
  &filereader.Token{Text: "abbess", Type: filereader.TextToken},
  &filereader.Token{Text: "abbei", Type: filereader.TextToken},
  &filereader.Token{Text: "abbei", Type: filereader.TextToken},
  &filereader.Token{Text: "abbomin", Type: filereader.TextToken},
  &filereader.Token{Text: "abbot", Type: filereader.TextToken},
  &filereader.Token{Text: "abbot", Type: filereader.TextToken},
  &filereader.Token{Text: "abbrevi", Type: filereader.TextToken},
  &filereader.Token{Text: "ab", Type: filereader.TextToken},
  &filereader.Token{Text: "abel", Type: filereader.TextToken},
  &filereader.Token{Text: "aberga", Type: filereader.TextToken},
  &filereader.Token{Text: "abergavenni", Type: filereader.TextToken},
  &filereader.Token{Text: "abet", Type: filereader.TextToken},
  &filereader.Token{Text: "abet", Type: filereader.TextToken},
  &filereader.Token{Text: "abhomin", Type: filereader.TextToken},
  &filereader.Token{Text: "abhor", Type: filereader.TextToken},
  &filereader.Token{Text: "abhorr", Type: filereader.TextToken},
  &filereader.Token{Text: "abhor", Type: filereader.TextToken},
  &filereader.Token{Text: "abhor", Type: filereader.TextToken},
  &filereader.Token{Text: "abhor", Type: filereader.TextToken},
  &filereader.Token{Text: "abhorson", Type: filereader.TextToken},
  &filereader.Token{Text: "abid", Type: filereader.TextToken},
  &filereader.Token{Text: "abid", Type: filereader.TextToken},
  &filereader.Token{Text: "abil", Type: filereader.TextToken},
  &filereader.Token{Text: "abil", Type: filereader.TextToken},
  &filereader.Token{Text: "ions", Type: filereader.TextToken},
}

var input = `aaron abaissiez abandon abandoned abase abash abate
abated abatement abatements abates abbess abbey abbeys abbominable
abbot abbots abbreviated abed abel aberga abergavenny abet abetting
abhominable abhor abhorr abhorred abhorring abhors abhorson abide
abides abilities ability ions`

func init() {

  AddTestCase("porter",TestCase{
    NewPorterFilter,
    input,
    expected,
  })

}

