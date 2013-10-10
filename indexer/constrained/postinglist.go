package constrained

import index "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "fmt"
/*import "strconv"*/
import "io"
import log "github.com/cihub/seelog"
import "bufio"
import "bytes"

type DatastoreTag string

func (t DatastoreTag) String() string {
return string(t)
}

type PostingListSet struct {
    Tag DatastoreTag
    listMap map[string]index.PostingList
    pl_init index.PostingListInitializer
    pl_entry_init func(id filereader.DocumentId) index.PostingListEntry

    Size int
    size_needs_refresh bool
}


func NewPostingListSet(tag DatastoreTag,
                       init index.PostingListInitializer) *PostingListSet{
    pls := new(PostingListSet)
    pls.listMap = make(map[string]index.PostingList)
    pls.Tag = tag
    pls.pl_init = init
    pls.pl_entry_init = init().EntryFactory
    return pls
}

func (pls PostingListSet) String() string {
    buf := new(bytes.Buffer)
    buf.WriteString(pls.Tag.String())
    buf.WriteString(fmt.Sprintf(" [%d entries, %d terms]", pls.Size,
    len(pls.listMap)))
    /*for term, _ := range pls.listMap {*/
        /*buf.WriteString(term + " ")*/
    /*}*/
    /*buf.WriteByte(']')*/

    return buf.String()
}



//Get and return the PostingList for a particular term
func (pls *PostingListSet) Get(term string) index.PostingList {
    pls.size_needs_refresh = true

    if pl, ok := pls.listMap[term]; ok {
        log.Debugf("Have posting list for %s.", term)
        return pl
    } else {
        pl = pls.pl_init()
        pls.listMap[term] = pl
        log.Debugf("Don't have posting list for %s. ", term)
        return pl
    }
}

func (pls *PostingListSet) Dump(w io.Writer) {
  var (
    pl index.PostingList
    it index.PostingListIterator
    term string
  )
    writer := bufio.NewWriter(w)

    for term, pl = range pls.listMap {

        for it = pl.Iterator(); it.Next(); {
        writer.WriteString(term)
        writer.WriteByte(' ')
        writer.WriteString(it.Value().Serialize())
        writer.WriteByte('\n')
        }
    }
    writer.Flush()
}

func (pls *PostingListSet) Load(r io.Reader) {
  var (
    pl index.PostingList
    pl_entry index.PostingListEntry
    ok bool
    term, parsed_term string
    parsed int
    /*docId int64*/
    e error
  )


    scanner := bufio.NewScanner(r)
    scanner.Split(bufio.ScanLines)
    for scanner.Scan() {
        if len(scanner.Text()) == 0 {
          continue
        }
        pl_entry = pls.pl_entry_init(0)

        parsed, e = fmt.Sscanln(scanner.Text(), &parsed_term, pl_entry)
        if e != nil {
          panic(e)
        }

        if parsed != 2 {
          pl_entry = nil
          continue
        }

        // Lookup PL, otherwise save the cost
        if term != parsed_term {

          term = parsed_term
          if pl, ok = pls.listMap[term]; !ok {
            pl = pls.pl_init()
            pls.listMap[term] = pl
          }
        }

        pl.InsertCompleteEntry(pl_entry)
        pls.Size++
    }
}

func (pls *PostingListSet) RecalculateLen() {
  var pl index.PostingList

    entries := 0
    for _, pl = range pls.listMap {
        entries += pl.Len()
    }
    pls.Size = entries
}
