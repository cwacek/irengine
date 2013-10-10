package indexer

import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import "io"
import "encoding/json"
import radix "github.com/cwacek/radix_go"

type LexiconInitializer func(datadir string, memLimit int) PostingList

type Lexicon interface {
	radix.RadixTree
	FindTerm([]byte) (LexiconTerm, bool)
	InsertToken(*filereader.Token)
	Print(io.Writer)
	SetPLInitializer(PostingListInitializer)
}

type PersistentLexicon interface {
	SaveToDisk()
	PrintDiskStats(io.Writer)
}

type TermFromTokenFunc func(*filereader.Token, PostingListInitializer) LexiconTerm

type LexiconTerm interface {
	Text() string
	Tf() int
	Df() int
	Idf(totalDocCount int) float64
	PostingList() PostingList
	Register(token *filereader.Token)
	String() string
	json.Marshaler
	/*json.Unmarshaler*/
}

type PostingListEntry interface {
	DocId() string
	Frequency() int
	Positions() []int
	String() string
	AddPosition(int)
	Serialize() string
	Deserialize([][]byte) error
}

type PostingListInitializer func() PostingList

type PostingList interface {
	GetEntry(id string) (PostingListEntry, bool)

	// Insert an entry into the posting list. Return true
	// If it creates a new PostingListEntry, false if it does
	// not (and just adds a position or something
	InsertEntry(token *filereader.Token) bool
	InsertRawEntry(text, docid string, pos int) bool
	InsertCompleteEntry(pl_entry PostingListEntry) bool

	String() string
	Len() int
	Iterator() PostingListIterator
	EntryFactory(docId string) PostingListEntry
}

type PostingListIterator interface {
	Next() bool
	Value() PostingListEntry
	Key() string
}

type Indexer interface {
	//Set the lexicon to the following
	Init(Lexicon) error

	// Add filters to use when reading terms into
	// the index
	AddFilter(f filters.Filter)

	//Human readable
	String() string

	//Print each term in the lexicon, along with
	// it's posting list
	PrintLexicon(r io.Writer)

	// Insert a document into the index
	Insert(t filereader.Document)

	// Give the number of indexed documents
	Len() int

	// This will block if insertions are occurring
	WaitInsert()

	// Delete the index from disk
	Delete()
}
