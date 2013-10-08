package constrained

import index "github.com/cwacek/irengine/indexer"
import "io"
import "strconv"
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

    sz int
    sz_needs_refresh bool
}


func NewPostingListSet(tag DatastoreTag,
                       init index.PostingListInitializer) *PostingListSet{
    pls := new(PostingListSet)
    pls.listMap = make(map[string]index.PostingList)
    pls.Tag = tag
    pls.pl_init = init
    return pls
}

func (pls PostingListSet) String() string {
    buf := new(bytes.Buffer)
    buf.WriteString(pls.Tag.String())
    buf.WriteString(" [")
    for term, _ := range pls.listMap {
        buf.WriteString(term + " ")
    }
    buf.WriteByte(']')

    return buf.String()
}



//Get and return the PostingList for a particular term
func (pls *PostingListSet) Get(term string) index.PostingList {
    pls.sz_needs_refresh = true

    if pl, ok := pls.listMap[term]; ok {
        log.Debugf("Have posting list for %s. Returning %v",
        term, pl.String())
        return pl
    } else {
        pl = pls.pl_init()
        pls.listMap[term] = pl
        log.Debugf("Don't have posting list for %s. " +
        "Returning a new one %v", term, pl.String())
        return pl
    }
}

func (pls *PostingListSet) Dump(w io.Writer) {
    writer := bufio.NewWriter(w)

    for term, pl := range pls.listMap {

        for it := pl.Iterator(); it.Next(); {
        writer.WriteString(term)
        writer.WriteByte(' ')
        writer.WriteString(it.Value().Serialize())
        writer.WriteByte('\n')
        }
    }
    writer.Flush()
}

func (pls *PostingListSet) Load(r io.Reader) {
    var pl index.PostingList
    var ok bool

    scanner := bufio.NewScanner(r)

    for scanner.Scan() {
        data := bytes.Fields(scanner.Bytes())
        if len(data) == 0 {
            continue
        }
        log.Debugf("Scanner read %v", data)

        if pl, ok = pls.listMap[string(data[0])]; !ok {
            pl = pls.pl_init()
            pls.listMap[string(data[0])] = pl
        }

        log.Debugf("Parsing positions from %s", string(data[2]))
        for _, position := range bytes.Split(data[2],[]byte{','}) {

            log.Debugf("Found position %v (%s)", position, string(position))

            posInt, err := strconv.Atoi(string(position))
            if err != nil {
                panic(err)
            }
            pl.InsertRawEntry(string(data[0]), string(data[1]),posInt)
        }

        log.Debugf("After insert, PL was %s", pl.String())
    }
}

func (pls *PostingListSet) Len() int {
    if !pls.sz_needs_refresh {
        return pls.sz
    }

    entries := 0
    for _, pl := range pls.listMap {
        entries += pl.Len()
    }
    pls.sz = entries
    pls.sz_needs_refresh = false
    return entries
}
