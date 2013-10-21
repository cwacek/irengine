package constrained

import index "github.com/cwacek/irengine/indexer"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/scanner/filereader"
import "os"
import "sync"
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

type LRUSet []*PLSContainer

func (l LRUSet) LeastRecent() *PLSContainer{
    if len(l) == 0 {
        return nil
    }
	return l[0]
}

func (l LRUSet) RemoveOldest() LRUSet {
    return l[1:]
}

type PLSContainer struct {
  Tag DatastoreTag
  Size int
  Hits int
  Dumps int
  Loads int
  PLS *PostingListSet
}

func NewPLSContainer(newPLS *PostingListSet) *PLSContainer {
  container := new(PLSContainer)
  container.Size = newPLS.Size
  container.Dumps = 0
  container.Loads = 0
  container.Hits = 1
  container.PLS = newPLS
  container.Tag = newPLS.Tag

  return container
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

func (lex *lexicon) IncrementPLSSize(tag DatastoreTag ) {

  if container, ok := lex.pl_set_cache[tag]; ok {
    container.Size++
  }

}

func (t *persistent_term) Register(token *filereader.Token) {
  pls := t.lex.RetrievePLS(t)
  pl := pls.Get(t.Text_)
  log.Debugf("Registering %s in posting list for %s",token, t.Text_)
  if pl.InsertEntry(token) {
      t.lex.IncrementPLSSize(pls.Tag)
      pls.Size++
      t.lex.currentLoad++
      /*log.Infof("Added term, incrementing current load to %d", t.lex.currentLoad)*/
  }

  log.Tracef("After registering, pls is %s.", pls.String())

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

  pl_set_cache               map[DatastoreTag]*PLSContainer
  lru_cache                  LRUSet
  swapped_cache              LRUSet

 swapped_worker_q            chan *PLSContainer
 swapped_lock       *sync.RWMutex
 lru_lock         *sync.RWMutex

	DataDirectory              string

    stats                      map[PLSStat]int
}

var load_factor_called int64

func (lex *lexicon) load_factor() float64 {
  log.Trace("checking loadfactor")
    if lex.maxLoad < 0 {
        return 0.0
    }

    if len(lex.lru_cache) == 0 {
      /*log.Infof("LRU is empty, so setting currentLoad from %d to 0", lex.currentLoad)*/
      lex.currentLoad = 0
    }

    load := float64(lex.currentLoad) / float64(lex.maxLoad)

    if load > 10.0 {
      log.Criticalf("Exceeded max allowed load")
    }

    /*log.Infof("Load factor is now %0.2f with %d PLS in memory", load, len(lex.lru_cache))*/
    /*if len(lex.lru_cache) < 5 {*/
      /*for _, x:= range lex.lru_cache {*/
        /*log.Infof("Due to %s", x.PLS)*/
      /*}*/
    /*}*/
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

	lex.PLInit = index.BasicPostingListInitializer

	lex.maxLoad = maxMem
	lex.currentLoad = 0
    if lex.maxLoad > 0 {
      switch {
      case maxMem > 20000:
        lex.perPLSLoad = 5000 //maxMem / 10
      default:
        lex.perPLSLoad = maxMem / 5
      }
    } else {
        // This is int max
        lex.perPLSLoad = int(^uint(0) >> 1)
    }

    if lex.perPLSLoad <= 10 {
        log.Criticalf("Warning. PLS Load is set very low (< 10) terms per PLS")
    }

	lex.pl_set_cache = make(map[DatastoreTag]*PLSContainer)
	lex.lru_cache = make(LRUSet, 0)
	lex.swapped_cache = make(LRUSet, 0)

  lex.stats = make(map[PLSStat]int)

  lex.swapped_lock = new(sync.RWMutex)
  lex.lru_lock = new(sync.RWMutex)

  lex.swapped_worker_q = make(chan *PLSContainer)
  go lex.SwappedWorker()
	return lex
}

func (lex *lexicon) Location() string {
  return lex.DataDirectory + "/"
}

func (lex *lexicon) DSPath(tag DatastoreTag) string {
	return lex.DataDirectory + "/" + string(tag)
}

func (lex *lexicon) makeRecent(pls *PLSContainer) {
  /*log.Infof("Making %s recent in %v", pls.Tag, lex.lru_cache)*/
  if len(lex.lru_cache) == 1 {
    return
  }
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
    var bestPls *PLSContainer
    var pls *PLSContainer

    for i := 0; i < len(lex.lru_cache); i++ {
      //iterate over in memory ones (from back for most recently used)
      if pls = lex.lru_cache[i]; pls.Size < lex.perPLSLoad {
          log.Debugf("Choosing existing PLS %s because its load is only %d/%d", pls.Tag, pls.Size, lex.perPLSLoad)
          // This is in memory, and has space, use it.
          return pls.Tag
        }
    }

    lex.swapped_lock.RLock()
    for _, pls = range lex.swapped_cache {
      switch {
      case bestPls == nil && float64(pls.Size) < 0.95 * float64(lex.perPLSLoad):
          log.Debugf("Considering existing PLS %s because its load is only %d/%d", pls.Tag, pls.Size, lex.perPLSLoad)
        bestPls = pls

      case bestPls != nil && pls.Hits > bestPls.Hits && 
      float64(pls.Size) < 0.75 * float64(lex.perPLSLoad):
          log.Debugf("Considering existing PLS %s because its load is only %d/%d and it's loaded alot", pls.Tag, pls.Size, lex.perPLSLoad)
        // If it gets loaded alot, and it's not super full use this one
        bestPls = pls
      }
    }
    lex.swapped_lock.RUnlock()

      if bestPls == nil {
        log.Debugf("Generating new PLS")
        return DatastoreTag(generateTag())
      }

    return bestPls.Tag
}

func (lex *lexicon) AddPLS(newPLS *PostingListSet) {
    container := NewPLSContainer(newPLS)

    lex.pl_set_cache[newPLS.Tag] = container
    lex.lru_cache = append(lex.lru_cache, container)
    lex.currentLoad += newPLS.Size
    /*log.Infof("Added a PLS of size %d. Load is now %d", newPLS.Size, lex.currentLoad)*/
}

// Retrieve the PostingList set being used by :term:
// For slightly lower level operation that RetrievePostingList
func (lex *lexicon) RetrievePLS(term * persistent_term) *PostingListSet {

    log.Debugf("Received %s.", term)
    log.Tracef("PLS_cache: %v", lex.pl_set_cache)
    lex.stats[PLSFetches]++

    StartAgain:

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

      case ok && pls.PLS != nil :
        log.Debugf("Retrieving PLS for %s from cache", term)
        lex.stats[PLSHits]++
        pls.Hits++

        if pls.Size > lex.perPLSLoad  && pls.PLS.DocCount() > 1 {
          //If we're over size, and the doc count is more than 1
          // (i.e. this isn't just a term with a huge posting list
          term.DataTag = DatastoreTag(generateTag())
          log.Debugf("Moving %s to new PLS. Old one was: %s", term, pls.PLS)
          /*log.Infof("Container sz: %d PLS sz: %d", pls.Size, pls.PLS.Size)*/
          newPLS := NewPostingListSet(term.DataTag, lex.PLInit)

          moved := TransferPL(pls.PLS, newPLS, term.Text_)
          /*log.Infof("Transfered %d entries", moved )*/
          pls.Size -= moved
          lex.currentLoad -= moved
          /*log.Infof("Reduced currentLoad by %d of offset add. Now %d", moved, lex.currentLoad)*/

          lex.AddPLS(newPLS)
          lex.evict()
          lex.stats[PLSCreates]++
          /*log.Criticalf("Moving %s TO A NEW PLS", term.Text_)*/
          goto StartAgain
        }

        //Reorder the LRU entries to put this one first
        // (in the background)
        lex.makeRecent(pls)

        return pls.PLS

    default:
        //pls.PLS is nil, which means we have it, but we swapped it
        //to disk at som point
        log.Debugf("Retrieving PLS for %s from disk", term.Text_)

        if file, err := os.Open(lex.DSPath(term.DataTag)); err == nil {
            pls.PLS = NewPostingListSet(term.DataTag, lex.PLInit)
            pls.Size = pls.PLS.Load(file)
            pls.Loads++
            lex.stats[PLSLoadCount]++
        log.Debugf("Read %s from %s", pls.Tag,
            lex.DSPath(term.DataTag))

            lex.swapped_worker_q <- pls

            file.Close()

        } else {
            panic(err)
        }

        lex.evict()
        return pls.PLS
    }
    return nil

}

//Remove the element from the swapped cache. No harm if we're slow, since 
// ALl that can happen is we choose a loaded PLS
func (lex *lexicon) SwappedWorker() {
  var pls, target *PLSContainer
  var i int

  for target = range lex.swapped_worker_q {

    lex.swapped_lock.Lock()
    switch target.PLS {
    case nil:
      //We're supposed to add it  beause it's swapped
      /*log.Infof("asked to swap %p. Swap cache is %v, LRU is %v", target, lex.swapped_cache, lex.lru_cache)*/
      lex.swapped_cache = append(lex.swapped_cache, target)

    default:
      //remove it
      /*log.Debugf("asked to mark %p as not swapped with swapped cache %v", target, lex.swapped_cache)*/
      for i, pls = range lex.swapped_cache {
        if pls == target {
          switch {
          case len(lex.swapped_cache) == 1:
            lex.swapped_cache = make([]*PLSContainer, 0)

          case i == 0:
            //WE're first
            lex.swapped_cache = lex.swapped_cache[i+1:]

          case i < len(lex.swapped_cache) - 1:
            // We're in the middle
            lex.swapped_cache = append(lex.swapped_cache[:i-1], lex.swapped_cache[i+1:]...)
          default:
            // WE're last
            lex.swapped_cache = lex.swapped_cache[:i-1]
          }
        }
      }
    }
    lex.swapped_lock.Unlock()
  }
}

// Retrieve the posting list for :term:, loading
// from disk if necessary (and evicting others if necessary)
func (lex *lexicon) RetrievePostingList(term *persistent_term) index.PostingList {

    pls := lex.RetrievePLS(term)
    return pls.Get(term.Text_)
}

func(lex *lexicon) SaveToDisk() {
  log.Criticalf("Saving to disk. May take some time with %d PLSes", len(lex.pl_set_cache))
    var pls *PLSContainer
    var tag DatastoreTag

    for tag, pls = range lex.pl_set_cache {
      if pls.PLS == nil {
        pls.PLS = NewPostingListSet(tag, lex.PLInit)
        if file, err := os.Open(lex.DSPath(tag)); err == nil {
          pls.PLS.Load(file)
          file.Close()
        }
        lex.evict()
        /*lex.AddPLS(pls)*/
      }
      lex.dump_pls(pls)
    }
}

//Evict the least recent PostingListSet from the Lexicon if
//necessary
func (lex *lexicon) evict() {

    var oldest *PLSContainer
    evicted := 0

    for ; lex.load_factor() > 0.8; {
        /*log.Tracef("Evicting oldest from LRUSet %v", lex.lru_cache)*/

        oldest = lex.lru_cache.LeastRecent()
        if oldest == nil {
            return
        }

        if oldest == nil {
          panic("Lies")
        }

        lex.dump_pls(oldest)
        if oldest.Size != oldest.PLS.Size {
          prev := oldest.PLS.Size
          oldest.PLS.RecalculateLen()
          panic(fmt.Sprintf("Container has %d, while pls has %d (corrected: %d)", oldest.Size,prev, oldest.PLS.Size))
        }
        oldest.Size = oldest.PLS.Size
        oldest.Dumps++
        oldest.PLS = nil
        /*log.Info("Reducing load by %d because we evicted %v", oldest.Size, oldest)*/
        lex.currentLoad -= oldest.Size

        //Ask the worker to add it to the swapped cache
        lex.swapped_worker_q <- oldest

        // We only need to remove. The new one will be added
        lex.lru_cache = lex.lru_cache.RemoveOldest()
        log.Debugf("Evicting %p", oldest)
        /*log.Tracef("After eviction, LRUSet: %v", lex.lru_cache)*/
        evicted++
        lex.stats[PLSDumpCount]++
    }

    if evicted > 0 {
        log.Debugf("Evicted %d PLS to disk. Load: %f",
                  evicted, lex.load_factor())
    }
}

func (lex *lexicon) dump_pls(oldPLS *PLSContainer) {
	log.Debugf("Dumping %s to %s", oldPLS, lex.DSPath(oldPLS.Tag))
	if file, err := os.Create(lex.DSPath(oldPLS.Tag)); err == nil {
    if oldPLS.PLS == nil {
      panic(err)
    }
		oldPLS.PLS.Dump(file)
    file.Close()
	} else {
		panic(err)
	}
}

func (lex *lexicon) PrintDiskStats(w io.Writer) {
    for stat, val := range lex.stats {
        fmt.Printf("# %s: %d\n", stat, val)
    }
}
