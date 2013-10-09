package actions

import filereader "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/indexer/constrained"
import "errors"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "os"
import "fmt"
import "github.com/cwacek/irengine/indexer/filters"
import "flag"

func RunIndexer() *run_index_action {
    return new(run_index_action)
}

type run_index_action struct {
    Args

    stopWordList *string
    indexRoot *string
    maxMem *int
    indexType *string
}

func (a *run_index_action) Name() string {
    return "index"
}

func (a *run_index_action) DefineFlags(fs *flag.FlagSet) {
    a.AddDefaultArgs(fs)

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
}

func (a *run_index_action) SetupIndex() (indexer.Indexer, error) {

    //Setup the lexicon
    if err := os.MkdirAll(*a.indexRoot, 0775); err != nil {
        return nil, err
    }

    lexicon := constrained.NewLexicon(*a.maxMem, *a.indexRoot)
    index := new(indexer.SingleTermIndex)
    index.Init(lexicon)

    switch *a.indexType {
    case "single-term":
        //Set the initializer, then fall through
        lexicon.SetPLInitializer(indexer.NewBasicPostingList)
        index.AddFilter(filters.SingleTermFilterSequence)
        fallthrough

    case "single-term-positional":
        //This is the default PL Initializer, so we won't set it
        // However, all the single-terms use the same filters, so
        // set them up.
        if file, err := os.Open(*a.stopWordList); err != nil {
            return nil, errors.New("Invalid path for stopwordlist: " + *a.stopWordList)
        } else {
            index.AddFilter(filters.NewStopWordFilterFromReader(file))
        }

    default:
        log.Criticalf("Unknown index type: %s", *a.indexType)
        return nil, errors.New("Unknown index type: "+ *a.indexType)
    }


    return index, nil
}

func (a *run_index_action) Run() {
    var index indexer.Indexer
    var err error

    SetupLogging(*a.verbosity)

    //Setup document walkers
    docStream := make(chan filereader.Document)

    walker := new(DocWalker)
    walker.WalkDocuments(*a.docroot, *a.docpattern, docStream)

    if index, err = a.SetupIndex(); err != nil {
        log.Criticalf("Error creating index: %v", err)
        return
    }


    /*// For each document.*/
    for doc := range docStream {
        index.Insert(doc)

        break
    }

    index.WaitInsert()

    fmt.Println(index.String())
    index.PrintLexicon(os.Stdout)
}


