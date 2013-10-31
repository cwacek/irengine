package query_engine

import log "github.com/cihub/seelog"
import "strings"
import "github.com/cwacek/irengine/scanner/filereader"

type Query struct {
	Id     string
	Text   string
	Engine string
}

func (q *Query) TokenizeToChan(out chan *filereader.Token) {

	var (
		token *filereader.Token
		ok    error
	)

	tokenizer := filereader.BadXMLTokenizer_FromReader(strings.NewReader(q.Text))
	log.Debugf("Created tokenizer")

	for {
		token, ok = tokenizer.Next()

		if ok != nil {
			break
		}
		log.Tracef("Pushing '%v' into output channel %v", token, out)

		out <- token
	}
	out <- &filereader.Token{Type: filereader.NullToken, DocId: 0, Position: 0, Final: true}
	log.Debugf("Done tokenizing")
}
