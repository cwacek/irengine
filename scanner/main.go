package main

import "github.com/cwacek/irengine/scanner/actions"
import "github.com/cwacek/subcommand"
import log "github.com/cihub/seelog"

func main() {
	defer log.Flush()
	Run()
}

func Run() {
	actions.SetupLogging(0)

	subcommand.Parse(true,
		actions.PrintTokens(),
		actions.RunIndexer(),
		actions.QueryEngineRunner(),
	)
}
