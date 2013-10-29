package query_engine

import log "github.com/cihub/seelog"
import "strings"
import "github.com/cwacek/irengine/scanner/filereader"

type Query struct {
	Text string
}

func (q *Query) TokenizeToChan(out chan *filereader.Token) {

	var (
		token *filereader.Token
		ok    error
	)

	tokenizer := filereader.BadXMLTokenizer_FromReader(strings.NewReader(q.Text))
	log.Info("Created tokenizer")

	for {
		log.Infof("Going to tokenize")
		token, ok = tokenizer.Next()

		if ok != nil {
			log.Infof("Done")
			break
		}
		log.Infof("Pushing '%v' into output channel %v", token, out)

		out <- token
	}
	out <- &filereader.Token{Type: filereader.NullToken, DocId: 0, Position: 0, Final: true}
	log.Infof("Done tokenizing")
}
