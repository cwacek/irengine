package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "math"
import "sort"

type DirichletQL struct {
	mu float64
}

func init() {
	RegisterRankingEngine("LM", &DirichletQL{0})
}

func (lm *DirichletQL) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex) *Response {

	var (
		q_term    *filereader.Token
		docScores = make(map[filereader.DocumentId]float64)
		term      indexer.LexiconTerm
		ok        bool
		pl        indexer.PostingList
		pl_iter   indexer.PostingListIterator
		pl_entry  indexer.PostingListEntry
		doc_info  *indexer.StoredDocInfo
	)

	avgDocLen := float64(index.TermCount()) / float64(index.DocumentCount)
	if lm.mu == 0 {
		lm.mu = avgDocLen
	}

	var partial_score, tf_d float64

	var i int
	for i, q_term = range query_terms {
		term, ok = index.Retrieve(q_term.Text)
		if !ok {
			continue
		}

		log.Debugf("Calculating score for query term %d: %s ",
			i, q_term.Text)

		// Iterate over the whole posting list
		pl = term.PostingList()
		for pl_iter = pl.Iterator(); pl_iter.Next(); {
			pl_entry = pl_iter.Value()
			tf_d = float64(pl_entry.Frequency())
			doc_info = index.DocumentMap[pl_entry.DocId()]

			log.Debugf("Obtained PL Entry %v with frequency %f", pl_entry, tf_d)

			partial_score = tf_d
			log.Debugf("Tf_d %f", tf_d)
			partial_score += lm.mu *
				(float64(term.Tf()) / float64(index.DocumentCount))
			log.Debugf("numerator: %f", partial_score)
			partial_score /= float64(doc_info.TermCount) + lm.mu
			log.Debugf("after division: %f. Log %f",
				partial_score, math.Log10(partial_score))

			docScores[pl_entry.DocId()] += math.Log10(partial_score)
		}
	}

	responseSet := NewResponse()
	for id, score := range docScores {
		doc_info = index.DocumentMap[id]

		log.Debugf("Doc: %s, Score: %0.4f", doc_info.HumanId, score)
		responseSet.Append(&Result{doc_info.HumanId, score})
	}

	sort.Sort(responseSet)
	return responseSet
}
