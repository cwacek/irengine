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

func QuerierRunner() *query_action {
	return new(query_action)
}

type query_action struct {
	Args

	queryFile *string

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

	a.host = fs.String("index.host", "localhost",
		"The host running the query engine")

	a.port = fs.Int("index.port", 10800,
		"The port on which the query engine can be found")
}

func (a *query_action) BufferQueriesFromFile(r io.Reader) {
	scanner := bufio.NewScanner(r)

	var q_id, line string
	var query *query_engine.Query

	for i := 0; scanner.Scan(); i++ {
		line = scanner.Text()
		switch {

		case strings.HasPrefix(line, "<num>"):
			q_id = strings.TrimPrefix(line, "<num> Number: ")

		case strings.HasPrefix(line, "<topic>"):

			if q_id == "" {
				log.Criticalf("Parsed query topic before identifier on line %d", i)
				continue
			}

			query = &query_engine.Query{Id: q_id,
				Text: strings.TrimPrefix(line, "<topic> Topic: ")}

			a.queryBuffer = append(a.queryBuffer, query)
			q_id = ""

		}
	}
}

func ZMQConnect(host string, port int) (sock *zmq.Socket, err error) {

	log.Debug("Setting up socket")
	if sock, err = zmq.NewSocket(zmq.REQ); err == nil {
		return
	}

	log.Infof("Connecting to server...")
	if err = sock.Connect(fmt.Sprintf("tcp://%s:%d", host, port)); err != nil {
		return
	}

	return
}

func (a *query_action) Run() {
	var (
		err       error
		requester *zmq.Socket
		asJSON    []byte
		response  query_engine.Response
	)

	SetupLogging(*a.verbosity)

	if *a.queryFile == "" {
		log.Criticalf("queryfile is required argument")
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
	}
	defer requester.Close()

	for query := range a.queryBuffer {

		if asJSON, err = json.Marshal(query); err == nil {
			log.Infof("Sending %s", asJSON)
			requester.SendBytes(asJSON, 0)
		}

		// Wait for reply:
		if reply, err := requester.RecvBytes(0); err == nil {
			log.Infof("Received %s", reply)
			err = json.Unmarshal(reply, &response)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}

	}
}
