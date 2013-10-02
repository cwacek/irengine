package main

import "flag"
import "os"
import "github.com/cwacek/irengine/scanner/actions"
import log "github.com/cihub/seelog"

var verbosity = 0

func main() {
	defer log.Flush()
	Run()
}

func Run() {

	readArgs()
	SetupLogging(verbosity)
	DoAction()

}

func readArgs() {
	docroot := flag.String("doc.root", "",
		`The root directory under which to find document`)

	flag.String("doc.pattern", `^[^\.].+`,
		`A regular expression to match document names`)

	flag.String("action", "",
		`Action:
      print_tokens    Just print all tokens in all processed documents
    `)

  flag.IntVar(&verbosity, "v", 0, "Be verbose [1, 2, 3]")

	flag.Parse()

	if *docroot == "" {
		log.Critical("Filepath is required")
		os.Exit(1)
	}

	log.Debugf("Read %s as filepath to search", *docroot)
}

func DoAction() {

	switch flag.Lookup("action").Value.String() {
	case "print_tokens":
		root := flag.Lookup("doc.root").Value.String()
    actions.PrintTokens(root)
	}
}
