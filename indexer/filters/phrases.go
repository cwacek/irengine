package filters

import log "github.com/cihub/seelog"
import "bytes"
import "github.com/cwacek/irengine/scanner/filereader"

func AugmentedTf(tf, max_tf int) float64 {
	return 0.5 + ((0.5 * float64(tf)) / float64(max_tf))
}

func NormalizedTf(tf, max_tf int) float64 {
	return float64(tf) / float64(max_tf)
}

type TerminateReason int

const (
	NonTerminal  TerminateReason = 0
	DiffPhraseId TerminateReason = iota
	StopWord
	MaxLen
)

type PhraseFilter struct {
	FilterPlumbing
	stopwords map[string]int
	limit     float64
	phraseLen int
	maxfreq   int
	termcount int

	tokenbuffer []*filereader.Token
}

//Build a new PhraseFilter which discards words
//with an augmented Tf more than limit, and returns
// phrases no more than phraseLen in length
func NewPhraseFilter(phraseLen int, limit float64) Filter {
	f := new(PhraseFilter)

	f.limit = limit
	f.phraseLen = phraseLen
	f.reset()

	f.Id = "phrases"
	f.self = f
	return f
}

func (f *PhraseFilter) reset() {
	f.tokenbuffer = make([]*filereader.Token, 0)
	f.stopwords = make(map[string]int)
	f.maxfreq = 0
	f.termcount = 0
}

func (f *PhraseFilter) endsPhrase(phrase []*filereader.Token,
	token *filereader.Token) TerminateReason {

	_, is_stop := f.stopwords[token.Text]
	switch {

	case is_stop:
		log.Tracef("Found stopword: %s", token.Text)
		return StopWord

	case len(phrase) == 0:
		return NonTerminal

	case phrase[0].PhraseId != token.PhraseId:
		return DiffPhraseId

	case len(phrase) >= f.phraseLen:
		return MaxLen

	default:
		return NonTerminal
	}
}

func (f *PhraseFilter) NotifyDocComplete() {

	for term, frequency := range f.stopwords {
		tf := NormalizedTf(frequency, f.maxfreq)
		log.Debugf("Calculated TF of %f for %s", tf, term)
		if tf < f.limit {
			delete(f.stopwords, term)
		}
	}

	log.Debugf("Stopwords: %v", f.stopwords)

	position_counter := 0
	/*text := new(bytes.Buffer)*/
	/*phrase_begin := f.tokenbuffer[0]*/
	var phrase *filereader.Token

	start_idx := 0
	end_idx := 0
	//find start
	var phrase_terms = f.tokenbuffer[start_idx:end_idx]

	for i, token := range f.tokenbuffer {

		log.Tracef("Before: (%d, %d) %v", start_idx, end_idx, phrase_terms)
		switch f.endsPhrase(phrase_terms, token) {
		case StopWord:
			log.Tracef("Found stopword %s", token.Text)
			phrase = makePhrase(phrase_terms, position_counter)
			if phrase != nil {
				f.Send(phrase)
				position_counter++
			}
			start_idx = i + 1
			end_idx = start_idx

		case DiffPhraseId:
			log.Tracef("Found new phraseid %s", token.Text)
			phrase = makePhrase(phrase_terms, position_counter)
			if phrase != nil {
				f.Send(phrase)
				position_counter++
			}

			start_idx = i
			end_idx = start_idx + 1

		case MaxLen:
			log.Tracef("Found maxlen %s", token.Text)
			phrase = makePhrase(phrase_terms, position_counter)
			if phrase != nil {
				f.Send(phrase)
				position_counter++
			}

			start_idx++
			fallthrough

		case NonTerminal:
			end_idx++
		}
		if start_idx == len(f.tokenbuffer) {
			break
		}
		if end_idx >= len(f.tokenbuffer) {
			end_idx = len(f.tokenbuffer) - 1
		}

		phrase_terms = f.tokenbuffer[start_idx:end_idx]
		log.Tracef("After: (%d, %d) %v", start_idx, end_idx, phrase_terms)
	}

	//Reset for the next document
	f.reset()
}

func makePhrase(tokens []*filereader.Token, position int) *filereader.Token {
	if len(tokens) == 0 {
		return nil
	}

	buf := new(bytes.Buffer)
	for i, tok := range tokens {
		if i > 0 {
			buf.WriteRune(' ')
		}
		buf.WriteString(tok.Text)
	}

	phrase := filereader.NewToken(buf.String(),
		filereader.TextToken)
	phrase.DocId = tokens[0].DocId
	phrase.Final = true
	phrase.Position = position

	return phrase
}

// Just store all the tokens. We'll actually deal with them
// once we're notified that the document is done.
func (f *PhraseFilter) Apply(tok *filereader.Token) []*filereader.Token {

	f.tokenbuffer = append(f.tokenbuffer, tok)
	f.stopwords[tok.Text] += 1
	if freq := f.stopwords[tok.Text]; freq > f.maxfreq {
		f.maxfreq = freq
	}

	return nil
}
