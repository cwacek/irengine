package actions

import "flag"
import log "github.com/cihub/seelog"
import "fmt"
import "github.com/cwacek/irengine/indexer"
import "os"
import "github.com/cwacek/irengine/indexer/constrained"
import "github.com/cwacek/irengine/query_engine"

func QueryEngineRunner() *query_engine_action {
	return new(query_engine_action)
}

type query_engine_action struct {
	Args

	indexRoot *string

	port *int
}

func (a *query_engine_action) Name() string {
	return "start-query-engine"
}

func (a *query_engine_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.indexRoot = fs.String("index.store", "/tmp/irengine",
		"The directory where a built index can be found.")

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

	indefinite_wait := make(chan int)
	<-indefinite_wait

	log.Info("Shutting down query-engine")
	engine.Stop()
}
