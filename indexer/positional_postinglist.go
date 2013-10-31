package indexer

import "sort"
import "errors"
import "fmt"
import "unicode"
import "io"
import "bytes"
import "strconv"
import "strings"
import "github.com/ryszard/goskiplist/skiplist"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"

var (
	BasicPostingListInitializer = PostingListInitializer{
		Name:       "basic",
		Positional: false,
		Create: func() PostingList {
			pl := new(positional_pl)
			pl.Length = 0
			pl.Positional = false
			pl.list = skiplist.NewCustomMap(DocumentIdLessThan)
			pl.entry_factory = NewBasicEntry
			return pl
		},
	}

	PositionalPostingListInitializer = PostingListInitializer{
		Name:       "positional",
		Positional: true,
		Create: func() PostingList {
			pl := new(positional_pl)
			pl.Length = 0
			pl.Positional = true
			pl.list = skiplist.NewCustomMap(DocumentIdLessThan)
			pl.entry_factory = NewPositionalEntry
			return pl
		},
	}
)

type pl_iterator struct {
	sk_iter skiplist.Iterator
}

func (it *pl_iterator) Value() PostingListEntry {
	return it.sk_iter.Value().(PostingListEntry)
}

func (it *pl_iterator) Next() bool {
	log.Trace("Calling Next()")
	cont := it.sk_iter.Next()
	log.Tracef("returned %v", cont)

	return cont
}

func (it *pl_iterator) Key() int {
	return it.sk_iter.Key().(int)
}

type positional_pl struct {
	list          *skiplist.SkipList
	Length        int
	Positional    bool
	entry_factory func(filereader.DocumentId) PostingListEntry
}

func (pl *positional_pl) IsPositional() bool {
	return pl.Positional
}

func (pl *positional_pl) FilterSequential(other PostingList,
	within int) PostingList {

	if !pl.Positional || !other.IsPositional() {
		panic(errors.New("FilterSequential requires positional posting lists"))
	}

	filtered := new(positional_pl)
	filtered.entry_factory = pl.entry_factory
	filtered.Length = 0
	filtered.Positional = true
	filtered.list = skiplist.NewCustomMap(DocumentIdLessThan)

	var plEntry, otherEntry, newEntry PostingListEntry
	var found bool

	for pl_iter := pl.Iterator(); pl_iter.Next(); {
		plEntry = pl_iter.Value()

		if otherEntry, found = other.GetEntry(plEntry.DocId()); !found {
			// if the document isn't even in our filtering list,
			// then just drop it.
			continue
		}

		newEntry = filtered.entry_factory(plEntry.DocId())

		plPos := plEntry.Positions()
		filterPos := otherEntry.Positions()
		fIdx := 0
		plIdx := 0

		for fIdx < len(filterPos) && plIdx < len(plPos) {
			// If the old position is within our range in the
			// filter, move the filter position up
			switch {

			case plPos[plIdx]+within >= filterPos[fIdx]:
				newEntry.AddPosition(filterPos[fIdx])
				fIdx++

			case plPos[plIdx]+within < filterPos[fIdx]:
				plIdx++

			default:
				log.Warnf("within: %d, plPos: %d @%d, filterPos: %d @%d",
					within,
					plPos[plIdx], plIdx, filterPos[fIdx], fIdx)
				panic("NOT ME")

			}
		}

		// If we 't actually keep any positions, store
		// the entry.
		if newEntry.Frequency() > 0 {
			filtered.InsertCompleteEntry(newEntry)
		}
	}

	return filtered
}

func (pl *positional_pl) EntryFactory(docid filereader.DocumentId) PostingListEntry {
	return pl.entry_factory(docid)
}

func DocumentIdLessThan(l, r interface{}) bool {
	return l.(filereader.DocumentId) < r.(filereader.DocumentId)
}

func (pl *positional_pl) Iterator() PostingListIterator {
	iter := new(pl_iterator)
	log.Trace("Creating new iterator")
	iter.sk_iter = pl.list.Iterator()
	return iter
}

func (pl *positional_pl) Len() int {
	return pl.Length
}

func (pl *positional_pl) GetEntry(id filereader.DocumentId) (PostingListEntry,
	bool) {
	log.Debugf("Looking for %d in posting list", id)

	if elem, ok := pl.list.Get(id); ok {
		log.Debugf("Found %#v", elem)
		return elem.(PostingListEntry), true
	}
	log.Debugf("Found nothing.")
	return nil, false
}

func (pl *positional_pl) InsertCompleteEntry(entry PostingListEntry) bool {
	pl.list.Set(entry.DocId(), entry)
	pl.Length++
	return true
}

func (pl *positional_pl) InsertEntry(token *filereader.Token) bool {
	log.Debugf("Inserting %s into posting list.", token)
	return pl.InsertRawEntry(token.Text, token.DocId, token.Position)
}

func (pl *positional_pl) InsertRawEntry(text string,
	docid filereader.DocumentId, position int) bool {

	if entry, ok := pl.GetEntry(docid); ok {
		//We have an entry for this doc, so we're adding a
		//position
		log.Debugf("%s exists. Adding position %d", docid, position)
		entry.AddPosition(position)
		return false
	}

	log.Debug("Creating new entry")
	entry := pl.entry_factory(docid)
	log.Tracef("Adding position %d to entry", position)
	entry.AddPosition(position)

	log.Trace("Inserting entry in posting list")
	pl.InsertCompleteEntry(entry)
	log.Trace("Complete")
	return true
}

func (pl positional_pl) String() string {
	entries := make([]string, 0)

	for i := pl.list.Iterator(); i.Next(); {
		entries = append(entries, i.Value().(PostingListEntry).Serialize())
	}
	return strings.Join(entries, " | ")
}

type basic_sk_entry struct {
	positional_sk_entry
	frequency int
}

func NewBasicEntry(docId filereader.DocumentId) PostingListEntry {
	entry := new(basic_sk_entry)
	entry.docId = docId
	return entry
}

func (p *basic_sk_entry) Serialize() string {
	return fmt.Sprintf("%d %d", p.docId, p.frequency)
}

func (p *basic_sk_entry) Scan(state fmt.ScanState, verb rune) error {
	var token []byte
	var e error
	var tmpInt int64

	//Scan the document id
	token, e = state.Token(true, unicode.IsDigit)
	if e != nil {
		return e
	}

	if tmpInt, e = strconv.ParseInt(string(token), 10, 64); e != nil {
		return e
	} else {
		p.docId = filereader.DocumentId(tmpInt)
	}

	state.SkipSpace()
	token, e = state.Token(true, unicode.IsDigit)
	if e != nil {
		return e
	}

	if tmpInt, e = strconv.ParseInt(string(token), 10, 32); e != nil {
		return e
	} else {
		p.frequency = int(tmpInt)
	}
	return nil
}

func (p *basic_sk_entry) Deserialize(enc [][]byte) error {
	var err error
	var tmp int64

	if tmp, err = strconv.ParseInt(string(enc[0]), 10, 64); err != nil {
		return err
	} else {
		p.docId = filereader.DocumentId(tmp)
	}

	if tmp, err = strconv.ParseInt(string(enc[1]), 10, 32); err != nil {
		return err
	} else {
		p.frequency = int(tmp)
	}

	return nil
}

func (p *basic_sk_entry) Frequency() int {
	return p.frequency
}

func (p *basic_sk_entry) Positions() []int {
	return make([]int, 0)
}

func (p *basic_sk_entry) AddPosition(pos int) {
	p.frequency++
}

func (p *basic_sk_entry) String() string {
	return fmt.Sprintf("(%s, %s)", p.docId, p.frequency)
}

var Space []byte = []byte{' '}

func (p *basic_sk_entry) SerializeTo(buf io.Writer) {
	docId := strconv.FormatInt(int64(p.docId), 10)

	io.WriteString(buf, docId)
	buf.Write(Space)
	docId = strconv.FormatInt(int64(p.frequency), 10)
}

type positional_sk_entry struct {
	docId     filereader.DocumentId
	positions []int
}

func NewPositionalEntry(docId filereader.DocumentId) PostingListEntry {
	entry := new(positional_sk_entry)
	entry.docId = docId
	entry.positions = make([]int, 0)
	return entry
}

func (p *positional_sk_entry) Scan(state fmt.ScanState, verb rune) error {
	var token []byte
	var e error
	var tmpInt int64

	//Scan the document id
	token, e = state.Token(true, unicode.IsDigit)
	if e != nil {
		return e
	}

	if tmpInt, e = strconv.ParseInt(string(token), 10, 64); e != nil {
		return e
	} else {
		p.docId = filereader.DocumentId(tmpInt)
	}

	for {
		token, e = state.Token(true, unicode.IsDigit)
		if len(token) == 0 || e == io.EOF {
			break
		}

		if tmpInt, e = strconv.ParseInt(string(token), 10, 32); e != nil {
			return e
		} else {
			p.AddPosition(int(tmpInt))
		}
	}
	return nil
}

func (p *positional_sk_entry) Deserialize(input [][]byte) error {
	var (
		position []byte
		posInt   int
		err      error
	)

	/*log.Debugf("Parsing positions from %s", string(input[1]))*/
	for _, position = range bytes.Split(input[1], []byte{','}) {

		/*log.Debugf("Found position %v (%s)", position, string(position))*/

		if posInt, err = strconv.Atoi(string(position)); err != nil {
			return err
		} else {
			p.AddPosition(posInt)
		}
	}

	return nil
}

func (p *positional_sk_entry) SerializeTo(buf io.Writer) {

	fmt.Fprintf(buf, "%d ", p.docId)

	if len(p.positions) == 0 {
		panic("positional entry has no positions")
	}

	for i, position := range p.positions {
		if i != 0 {
			fmt.Fprintf(buf, " %d", position)
		} else {
			fmt.Fprintf(buf, "%d", position)
		}
	}
}

func (p *positional_sk_entry) Serialize() string {
	buf := new(bytes.Buffer)

	buf.WriteString(fmt.Sprintf("%d", p.docId))
	buf.WriteRune(' ')

	for i, position := range p.positions {
		if i != 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(fmt.Sprintf("%d", position))
	}

	return buf.String()
}

func (p *positional_sk_entry) String() string {
	log.Tracef("Converting %#v positional_sk_entry to string", p)
	parts := make([]string, 0, len(p.positions)+2)
	/*posParts := make([]string, len(p.positions))*/

	parts = append(parts, strconv.FormatInt(int64(p.docId), 10))
	parts = append(parts, strconv.Itoa(len(p.positions)))

	log.Tracef("Writing PL entry: %#v", parts)
	return "(" + strings.Join(parts, ", ") + ")"
}

func (p *positional_sk_entry) Frequency() int {
	return len(p.positions)
}

func (p *positional_sk_entry) Positions() []int {
	return p.positions
}

func (p *positional_sk_entry) DocId() filereader.DocumentId {
	return p.docId
}

func (p *positional_sk_entry) AddPosition(pos int) {
	p.positions = append(p.positions, pos)
	sort.Ints(p.positions)
}
