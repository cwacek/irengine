package filters

import log "github.com/cihub/seelog"
import "regexp"
import "github.com/cwacek/irengine/scanner/filereader"
import "strings"
import "unicode"

var alpha_num = regexp.MustCompile(`([A-z]+)-([0-9]+)`)
var num_alpha = regexp.MustCompile(`([0-9]+)-([A-z]+)`)

type HyphenFilter struct {
	FilterPlumbing
}

func NewHyphenFilter(id string) Filter {
	f := new(HyphenFilter)
	f.Id = id
	f.self = f
	return f
}

func (f *HyphenFilter) IsDestructive() bool {
	return true
}

func (f *HyphenFilter) Apply(tok *filereader.Token) (res []*filereader.Token) {
	res = make([]*filereader.Token, 0, 2)

	var newtok *filereader.Token

	log.Debugf("Received '%s'", tok)

	if m := alpha_num.FindStringSubmatch(tok.Text); m != nil {
		// We have a match. Make a new token. If the alpha part
		// is longer than 3 characters, make it a separate token too.
		newtok = tok.Clone()
		log.Debugf("Split into %v", m)
		newtok.Text = m[1] + m[2]
		newtok.Final = true
		res = append(res, newtok)

		if len(m[1]) >= 3 {
			newtok = tok.Clone()
			newtok.Text = m[1]
			newtok.Final = true
			res = append(res, newtok)
		}

	} else if m := num_alpha.FindStringSubmatch(tok.Text); m != nil {

		newtok = tok.Clone()
		newtok.Text = m[1] + m[2]
		newtok.Final = true
		res = append(res, newtok)

		if len(m[2]) >= 3 {
			newtok = tok.Clone()
			newtok.Text = m[2]
			newtok.Final = true
			res = append(res, newtok)
		}

	} else {
		hyphenated := strings.FieldsFunc(tok.Text,
			func(r rune) bool { return unicode.Is(unicode.Hyphen, r) })

    if len(hyphenated) > 0 {

    }

		// This doesn't match, send it along.
		res = append(res, tok)
	}

	return
}

func isPrefix(chars string) bool{

  switch chars {

  default:
    return false

}
}
