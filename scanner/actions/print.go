package actions

import "fmt"
import "flag"
import filereader "github.com/cwacek/irengine/scanner/filereader"

func PrintTokens() *print_tokens_action {
	return new(print_tokens_action)
}

type print_tokens_action struct {
	Args

	docroot    *string
	docpattern *string

	workers      chan string
	output       chan string
	worker_count int
}

func (a *print_tokens_action) Name() string {
	return "print_tokens"
}

func (a *print_tokens_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.docroot = fs.String("doc.root", "",
		`The root directory under which to find document`)

	a.docpattern = fs.String("doc.pattern", `^[^\.].+`,
		`A regular expression to match document names`)

}

func (a *print_tokens_action) Run() {
	SetupLogging(*a.verbosity)

	docStream := make(chan filereader.Document)

	walker := new(DocWalker)
	walker.WalkDocuments(*a.docroot, *a.docpattern, docStream)

	for doc := range docStream {
		fmt.Printf("Document %s (%d tokens)\n", doc.Identifier(),
			doc.Len())
	}
}
