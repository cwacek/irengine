package indexer

import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import "io"
import "encoding/json"
import radix "github.com/cwacek/radix_go"

type Lexicon interface {
  radix.RadixTree
  FindTerm([]byte) (LexiconTerm, bool)
  InsertToken(*filereader.Token)
  Print(io.Writer)
  SetPLInitializer(PostingListInitializer)
}

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
}

type PostingList interface {
  GetEntry(id string) (PostingListEntry, bool)
  InsertEntry(token *filereader.Token) PostingListEntry
  String() string
  Len() int
  Iterator() PostingListIterator
}

type PostingListIterator interface {
  Next() bool
  Value() PostingListEntry
  Key() string
}

type PostingListInitializer func() PostingList

type Indexer interface {
  // Set the data directory to store index
  // files in and the memory limit. -1 means unlimited
  // memory
  Init(datadir string, memLimit int) error

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

  // Delete the index from disk
  Delete()
}
