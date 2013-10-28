package actions

import "flag"
import log "github.com/cihub/seelog"
import "fmt"
import "github.com/cwacek/irengine/indexer"
import "os"
import "github.com/cwacek/irengine/indexer/constrained"

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

}
