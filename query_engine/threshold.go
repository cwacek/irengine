package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/indexer"
import "sort"

type idf_sortable_tokens struct {
	tokens []*filereader.Token
	index  *indexer.SingleTermIndex
}

func (s idf_sortable_tokens) Len() int {
	return len(s.tokens)
}

func (s idf_sortable_tokens) Less(i, j int) bool {
	var (
		a, b indexer.LexiconTerm
		ok   bool
	)

	if a, ok = s.index.Retrieve(s.tokens[i].Text); !ok {
		// If we can't find it, it comes later
		return false
	}

	if b, ok = s.index.Retrieve(s.tokens[j].Text); !ok {
		// If we can't find it, it comes later
		return true
	}

	doccount := s.index.DocumentCount
	if indexer.Idf(a, doccount) < indexer.Idf(b, doccount) {
		return true
	} else {
		return false
	}
}

func (s idf_sortable_tokens) Swap(i, j int) {
	var tmp *filereader.Token

	tmp = s.tokens[i]
	s.tokens[i] = s.tokens[j]
	s.tokens[j] = tmp
}

func ThresholdQueryTerms(
	q_tokens []*filereader.Token,
	thresh float64,
	index *indexer.SingleTermIndex) [][]*filereader.Token {

	var (
		grouped_q_tokens = make([][]*filereader.Token, 2, 2)
		sorter           idf_sortable_tokens
	)

	sorter.tokens = q_tokens
	sorter.index = index

	sort.Sort(sorter)

	for i, token := range q_tokens {
		float_i := float64(i)
		if float_i/float64(len(q_tokens)) < thresh {
			log.Infof("Added %s at %0.2f to first group",
				token.Text, float_i/float64(len(q_tokens)))
			grouped_q_tokens[0] = append(grouped_q_tokens[0], token)
		}
		log.Infof("Added %s at %0.2f to second group",
			token.Text, float_i/float64(len(q_tokens)))
		grouped_q_tokens[1] = append(grouped_q_tokens[1], token)
	}

	return grouped_q_tokens
}
