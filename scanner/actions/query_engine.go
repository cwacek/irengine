package actions

import "flag"
import log "github.com/cihub/seelog"
import "fmt"
import zmq "github.com/pebbe/zmq3"
import "os"
import "strings"
import "github.com/cwacek/irengine/indexer/constrained"
import "github.com/cwacek/irengine/query_engine"

func QueryEngineRunner() *query_engine_action {
	return new(query_engine_action)
}

type deployed_engine struct {
	engine *query_engine.ZeroMQEngine
	socket *zmq.Socket
}

type query_engine_action struct {
	Args

	posRoot    *string
	singleRoot *string
	stemRoot   *string
	phraseRoot *string
	engineMap  map[string]deployed_engine

	port *int
}

func (a *query_engine_action) Name() string {
	return "start-query-engine"
}

func (a *query_engine_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.posRoot = fs.String("index.store.positional", "",
		"A directory containing a positional index")

	a.singleRoot = fs.String("index.store.single", "",
		"A directory containing a single-term index")

	a.stemRoot = fs.String("index.store.stem", "",
		"A directory containing a stemmed index")

	a.phraseRoot = fs.String("index.store.phrase", "",
		"A directory containing a phrase index")

	a.port = fs.Int("engine.port", 10800,
		"The port on which to listen for incoming queries")

	a.engineMap = make(map[string]deployed_engine)
}

// Load the index at path into our index set labeled as tag
func (a *query_engine_action) LoadEngine(tag, path string, port int) {

	if path == "" {
		return
	}

	index, err := constrained.SingleTermIndexFromDisk(path)

	if err != nil {
		log.Criticalf("Error loading index %s from disk: %v", tag, err)
	}

	engine := &query_engine.ZeroMQEngine{}
	engine.Init(index, port)

	go engine.Start()

	if socket, err := ZMQConnect("localhost", port); err != nil {
		panic(err)
	} else {
		a.engineMap[tag] = deployed_engine{engine, socket}
	}
	log.Infof("Loaded '%s' [%s]", tag, index.String())
}

func (a *query_engine_action) Run() {
	var (
		e        error
		socket   *zmq.Socket
		query    query_engine.Query
		deployed deployed_engine
		ok       bool
		tag      string
		response *query_engine.Response
	)

	SetupLogging(*a.verbosity)

	a.LoadEngine("single", *a.singleRoot, *a.port+1)
	a.LoadEngine("positional", *a.posRoot, *a.port+2)
	a.LoadEngine("stem", *a.stemRoot, *a.port+3)
	a.LoadEngine("phrase", *a.phraseRoot, *a.port+4)

	if len(a.engineMap) == 0 {
		log.Critical("One of the index.store arguments must be supplied")
		os.Exit(1)
	}

	log.Infof("Starting Dispatcher")

	if socket, e = zmq.NewSocket(zmq.REP); e != nil {
		log.Criticalf("Error: %v", e)
		os.Exit(1)
	}
	defer socket.Close()

	socket.Bind(fmt.Sprintf("tcp://*:%d", *a.port))

	for {
		log.Debugf("ZeroMQEngine waiting for messages")
		log.Flush()

		query_engine.ReceiveQuery(socket, &query)

		log.Infof("Query: %v", query)

		prefs := strings.Split(query.IndexPref, ",")
		for i, index := range prefs {
			tag = strings.TrimSpace(index)

			if deployed, ok = a.engineMap[tag]; !ok {
				log.Warnf("Don't know how to handle '%s'", tag)
				response = query_engine.ErrorResponse(
					"No loaded indexes could process prefs: " + query.IndexPref)
				continue
			}

			//If this is the last index, force it to answer
			if i == len(prefs)-1 {
				query.Force = true
			}
			log.Infof("Querying '%s'", tag)
			query.Send(deployed.socket)

			response = query_engine.ReceiveResponse(deployed.socket)
			response.Source = index

			if msg, isErr := response.IsError(); isErr {
				log.Warnf("Error from %s [%s]. Moving to next engine",
					tag, msg)
				continue
			} else {
				log.Infof("%s returned %d results", tag, len(response.Results))
			}

			// Break out so that we send with our current values
			break

		}

		log.Info("Returning results from " + tag)
		response.Send(socket)

	}

	log.Info("Shutting down query-engine")
	for _, deployed := range a.engineMap {
		deployed.engine.Stop()
	}
}
