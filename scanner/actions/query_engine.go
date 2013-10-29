package actions

import "flag"
import log "github.com/cihub/seelog"
import "fmt"
import "github.com/cwacek/irengine/indexer"
import "encoding/json"
import "os"
import "github.com/cwacek/irengine/indexer/constrained"
import "github.com/cwacek/irengine/query_engine"
import zmq "github.com/pebbe/zmq3"

func QueryEngineRunner() *query_engine_action {
	return new(query_engine_action)
}

type query_engine_action struct {
	Args

	indexRoot *string
	queryFile *string

	port *int
}

func (a *query_engine_action) Name() string {
	return "query-engine"
}

func (a *query_engine_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.indexRoot = fs.String("index.store", "/tmp/irengine",
		"The directory where a built index can be found.")

	a.queryFile = fs.String("bulk_queryfile", "",
		"A file containing a bunch of queries to run in bulk")

	a.port = fs.Int("engine.port", 10800,
		"The port on which to listen for incoming queries")
}

func (a *query_engine_action) Run() {
	var index *indexer.SingleTermIndex
	var err error
	var ranker query_engine.RelevanceRanker

	SetupLogging(*a.verbosity)

	if *a.indexRoot == "" {
		log.Critical("index.store is required argument")
		os.Exit(1)
	}

	index, err = constrained.SingleTermIndexFromDisk(*a.indexRoot)

	if err != nil {
		log.Criticalf("Error loading index from disk: %v", err)
		os.Exit(1)
	}

	fmt.Println("Loaded index: " + index.String())

	ranker = query_engine.NewCosineVSM()

	engine := &query_engine.ZeroMQEngine{}
	engine.Init(index, *a.port, ranker)
	go engine.Start()

	log.Infof("Connecting to server...")
	requester, _ := zmq.NewSocket(zmq.REQ)
	defer requester.Close()

	err = requester.Connect(fmt.Sprintf("tcp://localhost:%d", *a.port))

	if err != nil {
		panic(err)
	}

	var q query_engine.Query
	var response query_engine.Response

	for request_nbr := 99; request_nbr != 90; request_nbr-- {
		// send hello
		q = query_engine.Query{fmt.Sprintf("%d bottles of beer", request_nbr)}

		if asJSON, e := json.Marshal(q); e == nil {
			log.Infof("Sending %s", asJSON)
			requester.SendBytes(asJSON, 0)
		}

		// Wait for reply:
		if reply, e := requester.RecvBytes(0); e == nil {
			log.Infof("Received %s", reply)
			e = json.Unmarshal(reply, &response)
			if e != nil {
				panic(e)
			}
		} else {
			panic(e)
		}
	}

	log.Info("Shutting down query-engine")
	engine.Stop()
}
