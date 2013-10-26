package filters

import log "github.com/cihub/seelog"
import "strings"
import "github.com/cwacek/irengine/scanner/filereader"

type SlashFilter struct {
	FilterPlumbing
}

func NewSlashFilter(dummy string) Filter {
	f := new(SlashFilter)
	f.Id = "slashes"
	f.self = f
	return f
}

func (f *SlashFilter) Apply(tok *filereader.Token) []*filereader.Token {

	results := make([]*filereader.Token, 0)
	var newtok *filereader.Token

	parts := strings.Split(tok.Text, "/")

	if len(parts) > 1 {
		log.Debugf("Splitting %s into pieces", tok.Text)
		for _, part := range parts {

			newtok = CloneWithText(tok, part)
			results = append(results, newtok)
		}
	} else {

		results = append(results, tok)
	}
	return results
}
