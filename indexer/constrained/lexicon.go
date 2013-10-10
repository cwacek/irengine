package constrained

import index "github.com/cwacek/irengine/indexer"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/scanner/filereader"
import "os"
import "io"
import "fmt"
import "strconv"
import "math/rand"

type PLSStat int

const (
    PLSDumpCount PLSStat = iota
    PLSLoadCount
    PLSHits
    PLSCreates
    PLSFetches
)

func (T PLSStat) String() string {
    switch (T) {
    case PLSLoadCount:
        return "PLS Loads"
    case PLSDumpCount:
        return "PLS Dumps"
    case PLSHits:
        return "PLS Hits"
    case PLSFetches:
        return "PLS Fetches"
    case PLSCreates:
        return "PLS Creates"
    default:
        panic("Unknown stat type")
    }
}

type LRUSet []*PostingListSet

func (l LRUSet) LeastRecent() *PostingListSet {
    if len(l) == 0 {
        return nil
    }
	return l[0]
}

func (l LRUSet) RemoveOldest() LRUSet {
    return l[1:]
}

type persistent_term struct {
	index.Term

	lex     *lexicon
	DataTag DatastoreTag
}

func generateTag() string {
	tmpBytes := make([]byte, 0)
    log.Debugf("Appending random number like %d to tmpBytes",
    rand.Int63())
    tmpBytes = strconv.AppendInt(tmpBytes, rand.Int63(), 16)
    return string(tmpBytes[:12])
}

func NewTerm(tok *filereader.Token, lex *lexicon, tag DatastoreTag) index.LexiconTerm {
	term := new(persistent_term)
	term.Text_ = tok.Text
	term.Tf_ = 0 // because we increment with Register
	term.Pl = nil

	term.lex = lex
    log.Debug("Built half of term")

    term.DataTag = tag

	term.Register(tok)
	log.Debugf("Created term: %#v", term)
	return term
}

func (t *persistent_term) Register(token *filereader.Token) {
  log.Tracef("LRU_CACHE: %v", t.lex.lru_cache)
  pls := t.lex.RetrievePLS(t)
  log.Tracef("LRU_CACHE after RetrievePLS: %v", t.lex.lru_cache)
  pl := pls.Get(t.Text_)
  log.Debugf("Registering %s in posting list for %s",token, t.Text_)
  if pl.InsertEntry(token) {
      pls.Size++
      t.lex.pls_size_cache[pls.Tag]++
      t.lex.currentLoad++
  }

  log.Tracef("After registering, pls is %s. LRU_CACHE: %v", pls.String(), t.lex.lru_cache)

  t.Tf_ += 1

}

func (t *persistent_term) PostingList() index.PostingList {
	log.Debugf("Looking for posting list for %s", t.String())
	return t.lex.RetrievePostingList(t)
}

func (t persistent_term) String() string {
    return fmt.Sprintf("[%s %d @%s]", t.Text_, t.Tf_, t.DataTag)
}

func (t *persistent_term) Df() int {
	return t.PostingList().Len()
}

// Implements a Lexicon
type lexicon struct {
	index.TrieLexicon

	maxLoad, currentLoad, perPLSLoad int

	pl_set_cache               map[DatastoreTag]*PostingListSet
	lru_cache                  LRUSet
    pls_size_cache             map[DatastoreTag]int

	DataDirectory              string

    stats                      map[PLSStat]int
}

func (lex *lexicon) update_load() {
    lex.currentLoad = 0
    for _, pls := range lex.lru_cache {
        plsLoad := pls.Size
        lex.pls_size_cache[pls.Tag] = plsLoad
        lex.currentLoad += plsLoad
    }
}

func (lex *lexicon) load_factor() float64 {
    if lex.maxLoad < 0 {
        return 0.0
    }

    load := float64(lex.currentLoad) / float64(lex.maxLoad)
    log.Debugf("Load factor is now %0.2f with %d PLS in memory", load, len(lex.lru_cache))
    return load
}

func NewLexicon(maxMem int, dataDir string) index.Lexicon {
	var lex *lexicon
	lex = new(lexicon)
	lex.Init()

    if err := os.RemoveAll(dataDir); err != nil {
        panic(err)
    }

    if err := os.MkdirAll(dataDir, 0755); err != nil {
        panic(err)
    }

    lex.DataDirectory = dataDir


	// Wrap args
	lex.TermInit =
		func(tok *filereader.Token,
			p index.PostingListInitializer) index.LexiconTerm {
                term := NewTerm(tok, lex, lex.LeastUsedPLS())
                log.Debugf("Creating new term: %v", term)
                return term
		}

	lex.PLInit = index.NewPositionalPostingList

	lex.maxLoad = maxMem
	lex.currentLoad = 0
    if lex.maxLoad > 0 {
        lex.perPLSLoad = maxMem / 10
    } else {
        // This is int max
        lex.perPLSLoad = int(^uint(0) >> 1)
    }

    if lex.perPLSLoad <= 10 {
        log.Criticalf("Warning. PLS Load is set very low (< 10) terms per PLS")
    }

	lex.pl_set_cache = make(map[DatastoreTag]*PostingListSet)
	lex.pls_size_cache = make(map[DatastoreTag]int)
	lex.lru_cache = make(LRUSet, 0)

    lex.stats = make(map[PLSStat]int)

	return lex
}

func (lex *lexicon) DSPath(tag DatastoreTag) string {
	return lex.DataDirectory + "/" + string(tag)
}

func (lex *lexicon) makeRecent(pls *PostingListSet) {
  log.Debugf("Making %s recent", pls.Tag)
  /*log.Tracef("LRU_CACHE: %v", lex.lru_cache)*/
  for i, set := range lex.lru_cache {
    if set == pls {
      copy(lex.lru_cache[i:], lex.lru_cache[i+1:])
      log.Tracef("Rearranged at %d: %v", i, lex.lru_cache)
      lex.lru_cache[len(lex.lru_cache)-1] = set
      break
    }
  }
  /*log.Tracef("Afterwards: %v", lex.lru_cache)*/
}

// Find a PLS that's available
func (lex *lexicon) LeastUsedPLS() DatastoreTag {
    var bestPls DatastoreTag = ""

    for plsTag, sz := range lex.pls_size_cache {
        pls := lex.pl_set_cache[plsTag]
        switch {
        case pls != nil && sz < lex.perPLSLoad:
            log.Debugf("Choosing existing PLS %s because its load is only %d/%d",
            pls.Tag, sz, lex.perPLSLoad)
            // This is in memory, and has space, use it.
            bestPls = plsTag
            goto HaveBest

        case pls == nil && sz < lex.perPLSLoad:
            log.Debugf("Considering swapped PLS %s because its load is only %d/%d",
            plsTag, sz, lex.perPLSLoad)
            //Not in memory, but has space. Save it
            bestPls = plsTag
        }

    }
    if bestPls == "" {
    log.Debugf("Generating new PLS")
        bestPls = DatastoreTag(generateTag())
    }

    HaveBest:
    return bestPls
}

func (lex *lexicon) AddPLS(newPLS *PostingListSet) {
    lex.pls_size_cache[newPLS.Tag] = newPLS.Size
    lex.pl_set_cache[newPLS.Tag] = newPLS
    lex.lru_cache = append(lex.lru_cache, newPLS)
    lex.currentLoad += newPLS.Size
}

// Retrieve the PostingList set being used by :term:
// For slightly lower level operation that RetrievePostingList
func (lex *lexicon) RetrievePLS(term * persistent_term) *PostingListSet {

    log.Debugf("Received %s.", term)
    log.Tracef("PLS_cache: %v", lex.pl_set_cache)
    lex.stats[PLSFetches]++

	pls, ok := lex.pl_set_cache[term.DataTag]
    switch {
    case !ok:
        //We've never seen this one. Make a new one
        log.Debugf("Creating new PLS for %s", term)
        newPLS := NewPostingListSet(term.DataTag, lex.PLInit)
        lex.evict()
        lex.AddPLS(newPLS)
        lex.stats[PLSCreates]++
        return newPLS

    case ok && pls != nil :
        lex.stats[PLSHits]++
        log.Debugf("Retrieving PLS for %s from cache", term)
		//Reorder the LRU entries to put this one first
		// (in the background)
		go lex.makeRecent(pls)

		return pls

    default:
        //PLS is nil, which means we have it, but we swapped it
        //to disk at som point
        log.Debugf("Retrieving PLS for %s from disk", term)

        var newPLS *PostingListSet
        if file, err := os.Open(lex.DSPath(term.DataTag)); err == nil {
            newPLS = NewPostingListSet(term.DataTag, lex.PLInit)
            newPLS.Load(file)
            lex.stats[PLSLoadCount]++
        log.Debugf("Read %s from %s", newPLS,
            lex.DSPath(term.DataTag))
        } else {
            panic(err)
        }

        lex.evict()
        lex.AddPLS(newPLS)
        return newPLS
    }
    return nil

}

// Retrieve the posting list for :term:, loading
// from disk if necessary (and evicting others if necessary)
func (lex *lexicon) RetrievePostingList(term *persistent_term) index.PostingList {

    pls := lex.RetrievePLS(term)
    return pls.Get(term.Text_)
}

func(lex *lexicon) SaveToDisk() {

    for _, pls := range lex.lru_cache {
        lex.dump_pls(pls)
    }
}

//Evict the least recent PostingListSet from the Lexicon if
//necessary
func (lex *lexicon) evict() {

    evicted := 0

    for ; lex.load_factor() > 0.8; {
        log.Tracef("Evicting oldest from LRUSet %v", lex.lru_cache)

        oldest := lex.lru_cache.LeastRecent()
        if oldest == nil {
            return
        }

        lex.pls_size_cache[oldest.Tag] = oldest.Size
        lex.dump_pls(oldest)
        lex.pl_set_cache[oldest.Tag] = nil
        lex.currentLoad -= oldest.Size
        oldest = nil

        // We only need to remove. The new one will be added
        lex.lru_cache = lex.lru_cache.RemoveOldest()
        log.Tracef("After eviction, LRUSet: %v", lex.lru_cache)
        evicted++
        lex.stats[PLSDumpCount]++
    }

    if evicted > 0 {
        log.Debugf("Evicted %d PLS to disk. Load: %f",
                  evicted, lex.load_factor())
    }
}

func (lex *lexicon) dump_pls(oldPLS *PostingListSet) {
	log.Debugf("Dumping %s to %s", oldPLS, lex.DSPath(oldPLS.Tag))
	if file, err := os.Create(lex.DSPath(oldPLS.Tag)); err == nil {
		oldPLS.Dump(file)
	} else {
		panic(err)
	}
}

func (lex *lexicon) PrintDiskStats(w io.Writer) {
    for stat, val := range lex.stats {
        fmt.Printf("# %s: %d\n", stat, val)
    }
}
