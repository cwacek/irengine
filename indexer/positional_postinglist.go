package indexer

import "bytes"
import "sort"
import "strconv"
import "strings"
import "github.com/ryszard/goskiplist/skiplist"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

type positional_pl struct {
    list *skiplist.SkipList
}

func NewPositionalPostingList() PostingList {
    pl := new(positional_pl)
    pl.list = skiplist.NewStringMap()
    return pl
}

func (pl *positional_pl) Len() int {
    return pl.list.Len()
}

func (pl *positional_pl) GetEntry(id string) (PostingListEntry,
bool) {
    log.Debugf("Looking for %s in posting list", id)
    if elem, ok := pl.list.Get(id); ok {
        return elem.(*skiplist_entry), true
    }
    return  nil, false
}

func (pl *positional_pl) InsertEntry(token *filereader.Token) PostingListEntry {

    log.Debugf("Inserting %s into posting list.", token)

    if entry, ok := pl.GetEntry(token.DocId); ok {
        //We have an entry for this doc, so we're adding a
        //position
        log.Debugf("%s exists. Adding postion %d", token,
        token.Position)
        entry.AddPosition(token.Position)
        return entry
    }

    entry := NewPositionalEntry(token.DocId)
    entry.AddPosition(token.Position)

    pl.list.Set(entry.DocId(), entry)
    return entry
}

func (pl positional_pl) String() string {
    entries := make([]string,0)

    for  i := pl.list.Iterator(); i.Next(); {
        entries = append(entries,i.Value().(*skiplist_entry).String())
    }
    log.Debugf("Printing PL entries %v as '%s'", entries,
    strings.Join(entries, " "))
    return strings.Join(entries, " ")
}

type skiplist_entry struct {
    docId string
    positions []int
}

func NewPositionalEntry(docId string) PostingListEntry {
    entry := new(skiplist_entry)
    entry.docId = docId
    entry.positions = make([]int, 0)
    return entry
}

func (p *skiplist_entry) String() string {
    parts := make([]string, 0, len(p.positions) + 2)
    posParts := make([]string, len(p.positions))
    repr := new(bytes.Buffer)

    parts = append(parts, p.docId)
    parts = append(parts, strconv.Itoa(len(p.positions)))

    repr.WriteRune('{')
    for i,position := range p.positions {
        posParts[i] =  strconv.Itoa(position)
    }
    repr.WriteRune('}')

    parts = append(parts, "{" + strings.Join(posParts,",")+ "}")

    log.Debugf("Writing PL entry: %v", parts)
    return "(" + strings.Join(parts, ", ") + ")"
}

func (p *skiplist_entry) Frequency() int {
    return len(p.positions)
}

func (p *skiplist_entry) Positions() []int {
    return p.positions
}

func (p *skiplist_entry) DocId() string {
    return p.docId
}

func (p *skiplist_entry) AddPosition(pos int) {
    p.positions = append(p.positions, pos)
    sort.Ints(p.positions)
}
