package query_engine

import log "github.com/cihub/seelog"
import zmq "github.com/pebbe/zmq3"
import "strings"
import "encoding/json"
import "github.com/cwacek/irengine/scanner/filereader"

type Query struct {
	Id        string
	Text      string
	Engine    string
	IndexPref string
	Force     bool
}

func (q *Query) Send(s *zmq.Socket) {
	var msg []byte
	var e error

	if msg, e = json.Marshal(q); e != nil {
		panic(e)
	}
	s.SendBytes(msg, 0)
}

func ReceiveQuery(s *zmq.Socket, q *Query) {
	var msg []byte
	var e error

	if msg, e = s.RecvBytes(0); e != nil {
		panic(e)
	}

	log.Debugf("Received %s", msg)

	if e = json.Unmarshal(msg, q); e != nil {
		log.Criticalf("Error decoding JSON: %v", e)
		panic(e)
	}
}

func (q *Query) TokenizeToChan(out chan *filereader.Token) {

	var (
		token *filereader.Token
		ok    error
	)

	tokenizer := filereader.BadXMLTokenizer_FromReader(strings.NewReader(q.Text))
	log.Debugf("Created tokenizer")

	i := 1
	for {
		token, ok = tokenizer.Next()

		if ok != nil {
			break
		}
		log.Tracef("Pushing '%v' into output channel %v", token, out)
		token.Position = i
		i++

		out <- token
	}
	out <- &filereader.Token{Type: filereader.NullToken, DocId: 0, Position: 0, Final: true}
	log.Debugf("Done tokenizing")
}
