package main

import "github.com/cwacek/irengine/scanner/actions"
import "github.com/cwacek/subcommand"
import log "github.com/cihub/seelog"

//Add default args to fs

func main() {
	defer log.Flush()
	Run()
}

func Run() {
  subcommand.Parse(
    actions.PrintTokens(),
  )
}

