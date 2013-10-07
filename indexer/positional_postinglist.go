package indexer

import "sort"
import "strconv"
import "strings"
import "github.com/ryszard/goskiplist/skiplist"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

type pl_iterator struct {
    sk_iter skiplist.Iterator
}

func (it *pl_iterator) Value() PostingListEntry {
    return it.sk_iter.Value().(PostingListEntry)
}

func (it *pl_iterator) Next() bool {
    log.Debug("Calling Next()")
    cont := it.sk_iter.Next() 
    log.Debugf("returned %v", cont)

    return cont
}

func (it *pl_iterator) Key() string {
    return it.sk_iter.Key().(string)
}

type positional_pl struct {
    list *skiplist.SkipList
}

func NewPositionalPostingList() PostingList {
    pl := new(positional_pl)
    pl.list = skiplist.NewStringMap()
    return pl
}

func (pl *positional_pl) Iterator() PostingListIterator {
    iter := new(pl_iterator)
    log.Debug("Creating new iterator")
    iter.sk_iter = pl.list.Iterator()
    log.Debug("Done")
    return iter
}

func (pl *positional_pl) Len() int {
    return pl.list.Len()
}

func (pl *positional_pl) GetEntry(id string) (PostingListEntry,
bool) {
    log.Debugf("Looking for %s in posting list", id)
    if elem, ok := pl.list.Get(id); ok {
        log.Debugf("Found %#v", elem)
        return elem.(*skiplist_entry), true
    }
    log.Debugf("Found nothing.")
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

    log.Debugf("Creating new positional entry")
    entry := NewPositionalEntry(token.DocId)
    log.Debugf("Adding position %d to entry", token.Position)
    entry.AddPosition(token.Position)

    log.Debugf("Inserting entry in posting list")
    pl.list.Set(entry.DocId(), entry)
    log.Debugf("Complete")
    return entry
}

func (pl positional_pl) String() string {
    entries := make([]string,0)

    log.Debugf("Converting PL %#v to string", pl)
    for  i := pl.list.Iterator(); i.Next(); {
        entries = append(entries,i.Value().(*skiplist_entry).String())
    }
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
    log.Debugf("Converting %#v skiplist_entry to string", p)
    parts := make([]string, 0, len(p.positions) + 2)
    posParts := make([]string, len(p.positions))

    parts = append(parts, p.docId)
    parts = append(parts, strconv.Itoa(len(p.positions)))

    for i,position := range p.positions {
        posParts[i] =  strconv.Itoa(position)
    }

    parts = append(parts, "{" + strings.Join(posParts,",")+ "}")

    log.Debugf("Writing PL entry: %#v", parts)
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
