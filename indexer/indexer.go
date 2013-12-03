package indexer

import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import "io"
import "fmt"
import radix "github.com/cwacek/radix-go"

type LexiconInitializer func(datadir string, memLimit int) PostingList

type Lexicon interface {
	radix.RadixTree
	FindTerm([]byte) (LexiconTerm, bool)
	InsertToken(*filereader.Token) LexiconTerm
	Print(io.Writer)
	SetPLInitializer(PostingListInitializer)
	IsPositional() bool
}

type PersistentLexicon interface {
	SaveToDisk()
	LoadFromDisk(datadir string)
	PrintDiskStats(io.Writer)
	Location() string // Obtain the on disk location
}

type TermFromTokenFunc func(*filereader.Token, PostingListInitializer) LexiconTerm

type LexiconTerm interface {
	Text() string
	Tf() int
	PostingList() PostingList
	Register(token *filereader.Token)
	String() string
}

type PostingListEntry interface {
	DocId() filereader.DocumentId
	Frequency() int
	Positions() []int
	String() string
	AddPosition(int)
	Serialize() string
	SerializeTo(io.Writer)
	Deserialize([][]byte) error
	fmt.Scanner
}

type PostingListInitializer struct {
	Create     func() PostingList
	Name       string
	Positional bool
}

type PostingList interface {
	GetEntry(id filereader.DocumentId) (PostingListEntry, bool)

	// Insert an entry into the posting list. Return true
	// If it creates a new PostingListEntry, false if it does
	// not (and just adds a position or something
	InsertEntry(token *filereader.Token) bool
	InsertRawEntry(text string, docid filereader.DocumentId, pos int) bool
	InsertCompleteEntry(pl_entry PostingListEntry) bool

	String() string
	Len() int
	Iterator() PostingListIterator
	EntryFactory(docId filereader.DocumentId) PostingListEntry

	// tell whether or not the posting list is positional
	IsPositional() bool

	/* Remove the entries in <ids> from the posting list.
	 *  Return the number of entries removed. */
	Remove(ids ...filereader.DocumentId) int

	/* Filter this posting list against p, returning a posting
	 * list containing the positions for terms which occur less
	 * that 'within' positions after those in p. */
	FilterSequential(p PostingList, within int) PostingList
}

type PostingListIterator interface {
	Next() bool
	Value() PostingListEntry
	Key() int
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

	// Prune the posting lists according to some function
	Prune(pruner PostingListPruner)

	// Give the number of indexed documents
	Len() int

	// This will block if insertions are occurring
	WaitInsert()

	// Delete the index from disk
	Delete()
}
