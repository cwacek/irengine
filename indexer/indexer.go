package indexer

import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import "io"
import radix "github.com/cwacek/radix_go"

type Lexicon interface {
  radix.RadixTree
  FindTerm([]byte) (LexiconTerm, bool)
}

type LexiconTerm interface {
  Text() string
  Tf() int
  Df() int
  Idf(totalDocCount int) float64
  PostingList() PostingList
  Register(token *filereader.Token)
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
}

type PostingListInitializer func() PostingList

type StopWordList interface {
  Contains(string) bool
}

type StopWordFrequencyList interface {
  StopWordList
  Insert(string) int
  // Filter the StopWord list so that it only contains 
  // words for which the frequency is above freq
  Filter(freq float64)
}

type Indexer interface {
  // Set the data directory to store index
  // files in
  Init(datadir string) error

  // Add filters to use when reading terms into
  // the index
  AddFilter(f filters.Filter)

  //Tell the indexer to use a stopword list.
  //Note that  some indexers may not actually
  //respect this 
  SetStopWordList(sw StopWordList)

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
