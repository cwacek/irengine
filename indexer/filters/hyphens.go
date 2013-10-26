package filters

import log "github.com/cihub/seelog"
import "regexp"
import "github.com/cwacek/irengine/scanner/filereader"
import "strings"
import "unicode"

var alpha_num = regexp.MustCompile(`^([A-z]+)-([0-9]+)$`)
var num_alpha = regexp.MustCompile(`^([0-9]+)-([A-z]+)$`)
var WordPrefixes = map[string]bool{
	"anti":   true,
	"intra":  true,
	"re":     true,
	"co":     true,
	"macro":  true,
	"semi":   true,
	"de":     true,
	"micro":  true,
	"sub":    true,
	"hyper":  true,
	"non":    true,
	"supra":  true,
	"hypo":   true,
	"pre":    true,
	"trans":  true,
	"infra":  true,
	"pseudo": true,
	"un":     true,
}

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

	if m := alpha_num.FindStringSubmatch(tok.Text); m != nil {
		// We have a match. Make a new token. If the alpha part
		// is longer than 3 characters, make it a separate token too.
		newtok = tok.Clone()
		log.Tracef("Split into %v", m)
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
		justDigits := func(r rune) rune {
			switch {
			case unicode.IsDigit(r):
				return r
			default:
				return -1
			}
		}

		if len(strings.Map(justDigits, tok.Text)) > 0 {
			//There are digits in there
			res = append(res, tok)
			goto Exit
		}

		hyphenated := strings.FieldsFunc(tok.Text,
			func(r rune) bool { return unicode.Is(unicode.Hyphen, r) })

		switch len(hyphenated) {

		case 1:
			//No hyphens
			res = append(res, tok)

		case 2:
			if _, ok := WordPrefixes[hyphenated[0]]; ok {
				// include just prefixed and unprefixed
				res = append(res, CloneWithText(tok, hyphenated...))
				res = append(res, CloneWithText(tok, hyphenated[1]))

			} else {
				// Include both separatedly
				res = append(res, CloneWithText(tok, hyphenated[0]))
				res = append(res, CloneWithText(tok, hyphenated[1]))
			}

		default:
			for _, hyph_term := range hyphenated {
				res = append(res, CloneWithText(tok, hyph_term))
			}
			res = append(res, CloneWithText(tok, hyphenated...))
		}

	}

Exit:
	return
}

func isPrefix(chars string) bool {

	switch chars {

	default:
		return false

	}
}
