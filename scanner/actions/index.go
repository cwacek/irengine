package actions

import filereader "github.com/cwacek/irengine/scanner/filereader"
import "errors"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/indexer/constrained"
import "os"
import "fmt"
import "path/filepath"
import "runtime/pprof"
import "github.com/cwacek/irengine/indexer/filters"
import "flag"

func RunIndexer() *run_index_action {
	return new(run_index_action)
}

type run_index_action struct {
	Args

	docroot    *string
	docpattern *string

	stopWordList *string
	indexRoot    *string
	maxMem       *int
	indexType    *string

	phraseStop *float64
	phraseLen  *int

	cpuprofile *string
	memprofile *string
}

func (a *run_index_action) Name() string {
	return "index"
}

func (a *run_index_action) DefineFlags(fs *flag.FlagSet) {
	a.AddDefaultArgs(fs)

	a.docroot = fs.String("doc.root", "",
		`The root directory under which to find document`)

	a.docpattern = fs.String("doc.pattern", `^[^\.].+`,
		`A regular expression to match document names`)

	a.indexRoot = fs.String("index.store", "/tmp/irengine",
		"The directory in which to store the index")

	a.maxMem = fs.Int("index.memlimit", -1,
		"The maximum number of triples that can be loaded in to memory.")

	a.stopWordList = fs.String("index.stopwords", "",
		"A file containing stopwords to use.")

	a.indexType = fs.String("index.type", "single-term",
		`The type of index to build. Options:
    - single-term
    - single-term-positional
    - phrase
    - stemmed
    `)

	a.phraseStop = fs.Float64("phrase.limit", 0.2,
		"The relative term frequency required for a term to be considered a stop word")

	a.phraseLen = fs.Int("phrase.len", 2, "Maximum phrase length")

	a.cpuprofile = fs.String("cprofile", "", "write CPU profile to file")
	a.memprofile = fs.String("mprofile", "", "write memory profile to file")
}

func (a *run_index_action) SetupIndex() (indexer.Indexer, error) {

	lexicon := constrained.NewLexicon(*a.maxMem, *a.indexRoot)
	index := new(indexer.SingleTermIndex)
	index.Init(lexicon)

	switch *a.indexType {
	case "single-term":
		index.AddFilter(filters.SingleTermFilterSequence)
		lexicon.SetPLInitializer(indexer.BasicPostingListInitializer)

	case "single-term-positional":
		lexicon.SetPLInitializer(indexer.PositionalPostingListInitializer)
		index.AddFilter(filters.SingleTermFilterSequence)

	case "stemmed":
		lexicon.SetPLInitializer(indexer.BasicPostingListInitializer)
		index.AddFilter(filters.SingleTermFilterSequence)
		index.AddFilter(filters.Instantiate("porter"))

	case "phrase":
		lexicon.SetPLInitializer(indexer.BasicPostingListInitializer)
		if filter, err := filters.GetFactory("phrases"); err != nil {
			return nil, errors.New("Have no phrase filters. Cannot run phrase index")
		} else {
			filter.(*filters.PhraseFilterArgs).PhraseLen = *a.phraseLen
			filter.(*filters.PhraseFilterArgs).TfLimit = *a.phraseStop
			index.AddFilter(filter.Instantiate())
		}

	default:
		log.Criticalf("Unknown index type: %s", *a.indexType)
		return nil, errors.New("Unknown index type: " + *a.indexType)
	}

	// Allow anything to use the stopword list (even if it makes
	// no sense)
	if filter, err := filters.GetFactory("stopwords"); err != nil {
		log.Warnf("Not using stop word list")
	} else {
		if _, err := os.Lstat(*a.stopWordList); err == nil {
			log.Info("Using stopword list")

			path, err := filepath.Abs(*a.stopWordList)
			if err != nil {
				log.Criticalf("Couldn't turn '%s' into absolute path: %v", path, err)
				return nil, err
			}

			filter.(*filters.StopWordFilterFactory).Filename = path
			index.AddFilter(filter.Instantiate())
		} else {
			log.Criticalf("Couldn't read stop word list: %v", err)
			return nil, err
		}
	}

	return index, nil
}

func (a *run_index_action) Run() {
	var index indexer.Indexer
	var err error
	defer func() {
		log.Flush()
	}()

	SetupLogging(*a.verbosity)
	log.Info("Configured logging")

	//Setup document walkers
	docStream := make(chan filereader.Document)

	log.Info("Setting up document walker")
	if *a.docroot == "" {
		log.Criticalf("doc.root is required")
		os.Exit(1)
	}

	walker := new(DocWalker)
	walker.WalkDocuments(*a.docroot, *a.docpattern, docStream)

	if index, err = a.SetupIndex(); err != nil {
		log.Criticalf("Error creating index: %v", err)
		return
	}

	/*// For each document.*/
	ctr := 0
	for doc := range docStream {
		ctr++
		index.Insert(doc)

		if ctr > 1000 {

			if *a.cpuprofile != "" {
				f, err := os.Create(*a.cpuprofile)
				if err != nil {
					log.Critical(err)
					return
				}
				pprof.StartCPUProfile(f)
				defer pprof.StopCPUProfile()

				if ctr > 1100 {
					break
				}
			}

			if *a.memprofile != "" {
				f, err := os.Create(*a.memprofile)
				if err != nil {
					log.Critical(err)
					return
				}
				pprof.WriteHeapProfile(f)
				f.Close()
				*a.memprofile = ""
				break
			}
		}
	}

	index.WaitInsert()
	if ctr == 0 {
		log.Criticalf("No documents matched")
		return
	}

	log.Flush()
	fmt.Println(index.String())
	index.(*indexer.SingleTermIndex).Save()
	index.PrintLexicon(os.Stdout)
}
