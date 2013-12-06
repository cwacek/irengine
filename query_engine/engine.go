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
		msg                    []byte
		e                      error
		ok                     bool
		socket                 *zmq.Socket
		query                  Query
		resultSet              *Response
		filteredTokens         []*filereader.Token
		thresholdedQueryTokens [][]*filereader.Token
		ranker                 RelevanceRanker
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

		switch query.Type {
		case PhraseQuery:
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

			if query.QueryThresh < 1.0 {
				thresholdedQueryTokens = ThresholdQueryTerms(
					filteredTokens, query.QueryThresh, engine.index)
			} else {
				thresholdedQueryTokens = make([][]*filereader.Token, 1)
				thresholdedQueryTokens[0] = filteredTokens
			}

			log.Infof("Processing %d thresholded queries with %#v",
				len(thresholdedQueryTokens), ranker)

			resultSet = new(Response)
			for _, queryTermSet := range thresholdedQueryTokens {
				log.Infof("Querying with %v", queryTermSet)
				results := ranker.ProcessQuery(
					queryTermSet, engine.index, query.Force)
				log.Infof("Adding %d results to response", len(results.Results))
				resultSet.ExtendUnique(results)
				log.Infof("Response now had %d docs", len(resultSet.Results))

				break
				if resultSet.Len() > 100 {
					//If we have the number of documents we want without
					// processing more, return
					break
				}

			}

		case StatsQuery:
			resultSet = engine.LookupStats(query)
		}

		if msg, e = json.Marshal(resultSet); e != nil {
			panic(e)
		}

		socket.SendBytes(msg, 0)

	}
}

func (engine *ZeroMQEngine) LookupStats(query Query) *Response {
	if term, ok := engine.index.Retrieve(query.Text); !ok {
		return ErrorResponse(query.Text + " does not exist in index.")
	} else {

		response := NewResponse()
		response.Append(
			&Result{"IDF", indexer.Idf(term, engine.index.DocumentCount), ""})
		response.Append(
			&Result{"DF", float64(indexer.Df(term)), ""})
		response.Append(
			&Result{"Aggregate Tf", float64(term.Tf()), ""})
		response.Append(
			&Result{"PostingList", 0.0, term.PostingList().String()})

		return response
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
