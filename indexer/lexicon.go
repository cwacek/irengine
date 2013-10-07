package indexer

import "github.com/cwacek/irengine/scanner/filereader"
import radix "github.com/cwacek/radix_go"

// Implements a Lexicon
type TrieLexicon struct {
  radix.Trie
}

func (t *TrieLexicon) FindTerm(key []byte) (LexiconTerm, bool) {
  if elem, ok := t.Find(key); elem != nil {
    return elem.(LexiconTerm), ok
  }

  return nil, false
}

func NewTrieLexicon() Lexicon {
  lex := new(TrieLexicon)
  lex.Init()
  return lex
}

// Implements LexiconTerm
type Term struct {
  text string
  tf int
  pl PostingList
}

func NewTermFromToken(t *filereader.Token, p PostingListInitializer) *Term {
  term := new(Term)
  term.text = t.Text
  term.tf = 0 // because we increment with Register
  term.pl = p() // THis allows passing differnt types of posting lists.

  term.Register(t)
  return term
}

// Fulfill the RadixTreeEntry interface
func (t *Term) RadixKey() []byte {
  return []byte(t.text)
}

func (t *Term) Text() string {
  return t.text
}

func (t *Term) Register(token *filereader.Token) {
  t.pl.InsertEntry(token)
  t.tf += 1
}

func (t *Term) PostingList() PostingList {
  return t.pl
}

func (t *Term) Tf() int {
  return t.tf
}

func (t *Term) Df() int {
  return t.pl.Len()
}

func (t *Term) Idf(totalDocCount int) float64 {
  return (float64(totalDocCount) / float64(t.pl.Len()))
}
