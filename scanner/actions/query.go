package actions

import zmq "github.com/pebbe/zmq3"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/query_engine"
import "fmt"
import "os"
import "io"
import "flag"
import "bufio"
import "encoding/json"
import "strings"

func QueryRunner() *query_action {
	return new(query_action)
}

type query_action struct {
	Args

	queryFile *string
	engine    *string
	indexPref *string
	limit     *int

	host *string
	port *int

	queryBuffer []*query_engine.Query
}

func (a *query_action) Name() string {
	return "query"
}

func (a *query_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.queryFile = fs.String("queryfile", "",
		"A file containing a bunch of queries to run in bulk")

	a.engine = fs.String("ranking", "", `
  The ranking engine to use. Options are:
    COSINE    Cosine-normalized VSM similarity
    BM25      BM25 with Sparks-weight IDF
    LM        Query-likelihood with Dirichlet Smoothing`)

	a.limit = fs.Int("limit", 100,
		"Limit the results to this many results")

	a.host = fs.String("index.host", "localhost",
		"The host running the query engine")

	a.indexPref = fs.String("index.pref", "", `
  A comma separated list specifying the indices that
  should be used, in order of preference.`)

	a.port = fs.Int("index.port", 10800,
		"The port on which the query engine can be found")
}

func (a *query_action) BufferQueriesFromFile(r io.Reader) {
	scanner := bufio.NewScanner(r)

	var q_id, line string
	var query *query_engine.Query

	for i := 0; scanner.Scan(); i++ {
		line = scanner.Text()
		log.Debugf("Read %s from file", line)
		switch {

		case strings.HasPrefix(line, "<num>"):
			q_id = strings.TrimPrefix(line, "<num> Number: ")

		case strings.HasPrefix(line, "<title>"):

			if q_id == "" {
				log.Criticalf("Parsed query topic before identifier on line %d", i)
				continue
			}

			query = &query_engine.Query{
				Id:   q_id,
				Text: strings.TrimSpace(strings.TrimPrefix(line, "<title> Topic: ")),
			}

			a.queryBuffer = append(a.queryBuffer, query)
			q_id = ""

			log.Debugf("Created query %v. BUffer is %v", query, a.queryBuffer)
		}
	}
}

func ZMQConnect(host string, port int) (sock *zmq.Socket, err error) {

	log.Debug("Setting up socket")
	if sock, err = zmq.NewSocket(zmq.REQ); err != nil {
		return
	}

	log.Infof("Connecting to server...")
	if err = sock.Connect(fmt.Sprintf("tcp://%s:%d", host, port)); err != nil {
		return
	}

	return
}

func (a *query_action) Run() {
	defer log.Flush()
	var (
		err       error
		requester *zmq.Socket
		asJSON    []byte
		response  query_engine.Response
	)

	SetupLogging(*a.verbosity)

	switch {
	case *a.queryFile == "":
		log.Criticalf("queryfile is required argument")
		os.Exit(1)

	case *a.engine == "":
		log.Criticalf("ranking is required argument")
		os.Exit(1)

	case *a.indexPref == "":
		log.Criticalf("index.pref is required argument")
		os.Exit(1)
	}

	if file, err := os.Open(*a.queryFile); err == nil {
		a.BufferQueriesFromFile(file)

	} else {
		log.Criticalf("Failed to open query file: %v", err)
		os.Exit(1)
	}

	if requester, err = ZMQConnect(*a.host, *a.port); err != nil {
		log.Criticalf("Failed to connect socket: %v", err)
		return
	}
	defer requester.Close()

	for _, query := range a.queryBuffer {
		query.Engine = *a.engine
		query.IndexPref = *a.indexPref

		if asJSON, err = json.Marshal(query); err == nil {
			log.Debugf("Sending %s", asJSON)
			requester.SendBytes(asJSON, 0)
		} else {
			panic(err)
		}

		// Wait for reply:
		if reply, err := requester.RecvBytes(0); err == nil {
			log.Debugf("Received %s", reply)

			err = json.Unmarshal(reply, &response)
			switch {
			case err != nil:
				panic(err)

			case response.Error != "":
				log.Criticalf("Query failed: %s", response.Error)

			default:
				for i, result := range response.Results {
					fmt.Printf("%s Q0 %s %d %0.6f %s\n", query.Id, result.Document, i, result.Score, response.Source)
					if i >= *a.limit {
						break
					}
				}
			}

		} else {
			panic(err)
		}

	}
}