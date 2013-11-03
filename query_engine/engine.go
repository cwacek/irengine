package query_engine

import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/indexer"
import log "github.com/cihub/seelog"
import zmq "github.com/pebbe/zmq3"
import "fmt"
import "encoding/json"

type ZeroMQEngine struct {
	index  *indexer.SingleTermIndex
	ranker RelevanceRanker

	port    int
	control chan int

	filterStart chan *filereader.Token
	filterEnd   chan *filereader.Token
}

func (engine *ZeroMQEngine) Stop() {
	engine.control <- 1
}

// Wait for a signal on the control channel,
// then shut everything down
func (engine *ZeroMQEngine) watch_for_exit() {
	<-engine.control
	log.Infof("Shutting down.")

	if engine.filterStart != nil {
		close(engine.filterStart)
	}

	if engine.filterEnd != nil {
		close(engine.filterEnd)
	}
}

func (engine *ZeroMQEngine) Start() error {
	var (
		msg            []byte
		e              error
		ok             bool
		socket         *zmq.Socket
		query          Query
		resultSet      *Response
		filteredTokens []*filereader.Token
		ranker         RelevanceRanker
	)

	if socket, e = zmq.NewSocket(zmq.REP); e != nil {
		return e
	}
	defer socket.Close()

	log.Infof("Starting ZeroMQEngine")
	socket.Bind(fmt.Sprintf("tcp://*:%d", engine.port))

	for {
		log.Debugf("ZeroMQEngine waiting for messages")
		log.Flush()
		if msg, e = socket.RecvBytes(0); e != nil {
			panic(e)
		}

		log.Infof("Received %s", msg)

		if e = json.Unmarshal(msg, &query); e != nil {
			log.Criticalf("Error decoding JSON: %v", e)
			panic(e)
		}
		log.Infof("Decoded %v", query)

		if ranker, ok = RankingEngines[query.Engine]; !ok {

			if msg, e = json.Marshal(
				ErrorResponse("Unsupported ranking engine: " + query.Engine)); e != nil {
				panic(e)
			}

			socket.SendBytes(msg, 0)
			continue
		}

		query.TokenizeToChan(engine.filterStart)

		filteredTokens = engine.getDocTokens(engine.filterEnd)

		log.Infof("Processing query with %#v", ranker)
		resultSet = ranker.ProcessQuery(
			filteredTokens, engine.index, query.Force)

		if msg, e = json.Marshal(resultSet); e != nil {
			panic(e)
		}

		socket.SendBytes(msg, 0)
	}

}

func (engine *ZeroMQEngine) Init(index *indexer.SingleTermIndex, port int) error {

	engine.index = index
	engine.port = port
	engine.control = make(chan int)

	go engine.watch_for_exit()

	engine.filterStart = make(chan *filereader.Token, 100)
	engine.filterEnd = make(chan *filereader.Token, 100)

	go engine.index.FilterTokens(engine.filterStart, engine.filterEnd)

	return nil
}

// Read a documents worth of tokens from a channel and
// return it as a slice of tokens.
func (engine *ZeroMQEngine) getDocTokens(filterOut chan *filereader.Token) (out []*filereader.Token) {

	var token *filereader.Token
	out = make([]*filereader.Token, 0)

	for {
		token = <-filterOut

		if token.Type == filereader.NullToken {
			break
		}

		out = append(out, token)
	}

	return
}
