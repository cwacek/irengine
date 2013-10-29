package indexer

import "io"
import "encoding/json"
import "sort"
import "fmt"
import "math"
import "bytes"
import "github.com/cwacek/irengine/scanner/filereader"
import radix "github.com/cwacek/radix-go"
import log "github.com/cihub/seelog"

// Implements a Lexicon
type TrieLexicon struct {
	radix.Trie
	PLInit   PostingListInitializer
	TermInit TermFromTokenFunc
}

func (t *TrieLexicon) FindTerm(key []byte) (LexiconTerm, bool) {

	if elem, ok := t.Find(key); elem != nil {
		return elem.(LexiconTerm), ok
	}

	return nil, false
}

func (t *TrieLexicon) SetPLInitializer(pl_init PostingListInitializer) {
	if t.Len() > 0 {
		panic("Cannot set PL initializer after terms are inserted")
	}

	t.PLInit = pl_init
}

/* Insert a term in to the lexicon, and return the updated
 * term frequency for that term */
func (t *TrieLexicon) InsertToken(token *filereader.Token) LexiconTerm {

	if token.Type == filereader.NullToken {
		// This shouldn't get through, but ignore it if it does
		return nil
	}

	log.Debugf("Looking for %s in the lexicon.", token)
	var term LexiconTerm

	if term, ok := t.FindTerm([]byte(token.Text)); ok {
		// We found the term
		log.Debugf("Found %s in the lexicon: %s", token.Text, term.String())

		term.Register(token)
	} else {

		log.Tracef("Creating new term via %v", t.TermInit)
		term = t.TermInit(token, t.PLInit)

		log.Debugf("Created new term: %s. Inserting into lexicon", term.String())
		// Insert the new term
		t.Insert(term.(radix.RadixTreeEntry))
	}
	return term
}

func (t *TrieLexicon) Print(w io.Writer) {

	df_array := make([]int, 0, t.Len())
	dfSum := 0

	for i, entry := range t.Walk() {
		var term LexiconTerm

		defer func() {
			if x := recover(); x != nil {
				log.Criticalf("Error printing term %#v with posting list %#v: %v",
					term, term.PostingList(), x)
				log.Flush()
				panic(x)
			}
		}()

		term = entry.(LexiconTerm)
		log.Tracef("Walking found term %s", term.String())
		_, err := io.WriteString(w,
			fmt.Sprintf("%d. '%s' [%d]: %s\n",
				i+1, term.Text(),
				term.Tf(), term.PostingList()))

		if err != nil {
			panic(err)
		}

		dfSum += term.Df()
		df_array = append(df_array, term.Df())

	}

	sort.Ints(df_array)
	log.Debugf("Have %d term frequencies", len(df_array))

	statsFmt := `
  Term Count: %d
  Max DF:     %d
  Min DF:     %d
  Mean DF:    %0.2f
  Median DF:  %d
  `

	io.WriteString(w, fmt.Sprintf(statsFmt,
		t.Len(),
		df_array[len(df_array)-1],
		df_array[0],
		float64(dfSum)/float64(len(df_array)),
		df_array[len(df_array)/2]))

}

func NewTrieLexicon() Lexicon {
	lex := new(TrieLexicon)
	lex.Init()
	lex.PLInit = PositionalPostingListInitializer
	lex.TermInit = NewTermFromToken
	return lex
}

// Implements LexiconTerm
type Term struct {
	Text_ string
	Tf_   int
	Pl    PostingList
}

func NewTermFromToken(t *filereader.Token, p PostingListInitializer) LexiconTerm {
	term := new(Term)
	term.Text_ = t.Text
	term.Tf_ = 0         // because we increment with Register
	term.Pl = p.Create() // THis allows passing differnt types of posting lists.

	term.Register(t)
	log.Tracef("Created term: %#v", term)
	return term
}

// Fulfill the RadixTreeEntry interface
func (t *Term) RadixKey() []byte {
	return []byte(t.Text_)
}

func (t *Term) Text() string {
	return t.Text_
}

func (t *Term) Register(token *filereader.Token) {
	log.Debugf("Registering %s in Term", token)
	t.PostingList().InsertEntry(token)
	t.Tf_ += 1
	log.Debug("Registered")
}

func (t *Term) PostingList() PostingList {
	return t.Pl
}

func (t *Term) Tf_d(d filereader.DocumentId) float64 {
	pl_entry, ok := t.PostingList().GetEntry(d)
	if !ok {
		return 0.0
	}

	return float64(pl_entry.Frequency())
}

func (t *Term) Tf() int {
	return t.Tf_
}

func (t *Term) Df() int {
	return t.PostingList().Len()
}

func (t *Term) Idf(totalDocCount int) float64 {
	plLen := t.Df()
	log.Infof("%s has posting list length: %d", t.Text(), plLen)
	return 1 + math.Log10(float64(totalDocCount)/float64(plLen))
}

func (t Term) String() string {
	return fmt.Sprintf("['%s' %s]", t.Text_, t.PostingList().String())
}

func (t *Term) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	log.Debugf("Marshalling term %#v", t)
	for i := t.PostingList().Iterator(); i.Next(); {
		log.Debugf("Iterating with %#v", i)
		log.Debugf("Iterating over PL. Entry is %#v", i.Value())
		log.Flush()
		posList := i.Value().Positions()
		log.Debugf("Positions is %v", posList)
		if positions, err := json.Marshal(i.Value().Positions()); err != nil {
			log.Debug("Failed to marshal positions")
			return nil, err
		} else {
			buf.WriteString(fmt.Sprintf(`{"term": "%s", "doc": "%s", "tf": %d, "pos": %s}`,
				t.Text_, i.Value().DocId(), i.Value().Frequency(),
				positions))
		}
	}
	log.Debug("Finished iterating over PL")

	log.Debugf("buffer has %s", buf.String())
	ret := make([]byte, buf.Len())
	copy(ret, buf.Bytes())
	log.Debugf("Returning has '%s'", ret)
	return ret, nil
}
