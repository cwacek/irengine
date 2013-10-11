package actions

import filereader "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/indexer/constrained"
import "errors"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "os"
import "fmt"
import "runtime/pprof"
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

    phraseStop *float64
    phraseLen *int


    cpuprofile *string
    memprofile *string
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

    a.phraseStop = fs.Float64("phrase.limit", 0.2,
    "The relative term frequency required for a term to be considered a stop word")

    a.phraseLen = fs.Int("phrase.len", 2, "Maximum phrase length")

    a.cpuprofile = fs.String("cprofile", "", "write CPU profile to file")
    a.memprofile  = fs.String("mprofile", "", "write memory profile to file")
}

func (a *run_index_action) SetupIndex() (indexer.Indexer, error) {

    lexicon := constrained.NewLexicon(*a.maxMem, *a.indexRoot)
    index := new(indexer.SingleTermIndex)
    index.Init(lexicon)

    switch *a.indexType {
    case "single-term":
        index.AddFilter(filters.SingleTermFilterSequence)
        lexicon.SetPLInitializer(indexer.NewBasicPostingList)

    case "single-term-positional":
        lexicon.SetPLInitializer(indexer.NewPositionalPostingList)
        index.AddFilter(filters.SingleTermFilterSequence)

    case "stemmed":
        lexicon.SetPLInitializer(indexer.NewBasicPostingList)
        index.AddFilter(filters.SingleTermFilterSequence)
        index.AddFilter(filters.NewPorterFilter("porterstemmer"))

    case "phrase":
        lexicon.SetPLInitializer(indexer.NewBasicPostingList)
        index.AddFilter(
            filters.NewPhraseFilter(*a.phraseLen, *a.phraseStop))

    default:
        log.Criticalf("Unknown index type: %s", *a.indexType)
        return nil, errors.New("Unknown index type: "+ *a.indexType)
    }

    // Allow anything to use the stopword list (even if it makes
    // no sense)
    if file, err := os.Open(*a.stopWordList); err != nil {
        log.Warnf("Not using stop word list")
    } else {
        log.Info("Using stopword list")
        index.AddFilter(filters.NewStopWordFilterFromReader(file))
        file.Close()
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

    log.Flush()
    fmt.Println(index.String())
    index.PrintLexicon(os.Stdout)
    index.(*indexer.SingleTermIndex).Save()
}



