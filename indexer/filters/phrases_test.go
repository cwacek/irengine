package filters

import "github.com/cwacek/irengine/scanner/filereader"

var phrase_expected = []*filereader.Token{
	&filereader.Token{Text: "once", Type: filereader.TextToken},
	&filereader.Token{Text: "a young", Type: filereader.TextToken},
	&filereader.Token{Text: "young man", Type: filereader.TextToken},
	&filereader.Token{Text: "When", Type: filereader.TextToken},
	&filereader.Token{Text: "played with", Type: filereader.TextToken},
	&filereader.Token{Text: "with shoes", Type: filereader.TextToken},
	&filereader.Token{Text: "frequently", Type: filereader.TextToken},
	&filereader.Token{Text: "would wear", Type: filereader.TextToken},
	&filereader.Token{Text: "wear them", Type: filereader.TextToken},
	&filereader.Token{Text: "Tarnation", Type: filereader.TextToken},
}

var phrase_input = `I once was a young man. When I was, I played
with shoes; frequently I would wear them. Tarnation.`

func init() {

	builder := func(id string) Filter {
		return NewPhraseFilter(2, 0.4)
	}

	AddTestCase("phrases", TestCase{
		builder,
		phrase_input,
		phrase_expected,
	})

}
