package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "sort"

type BM25 struct {
	k1 float64
	k2 int
	b  float64
}

func init() {
	RegisterRankingEngine("BM25", &BM25{1.2, 1, 0.75})
}

func (bm *BM25) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex) *Response {

	var (
		q_term    *filereader.Token
		query_tf  = make(map[string]int)
		docScores = make(map[filereader.DocumentId]float64)
		term      indexer.LexiconTerm
		ok        bool
		pl        indexer.PostingList
		pl_iter   indexer.PostingListIterator
		pl_entry  indexer.PostingListEntry
		doc_info  *indexer.StoredDocInfo
	)

	for _, q_term = range query_terms {
		query_tf[q_term.Text]++
		log.Infof("Processing token %s. Have %d", q_term, query_tf[q_term.Text])
	}

	var partial_score, tf_d float64
	avgDocLen := float64(index.TermCount) / float64(index.DocumentCount)

	/* For each term in the query */
	for _, q_term = range query_terms {
		q_term_tf := query_tf[q_term.Text]
		term, ok = index.Retrieve(q_term.Text)
		if !ok {
			continue
		}

		// Iterate over the whole posting list
		pl = term.PostingList()
		for pl_iter = pl.Iterator(); pl_iter.Next(); {
			pl_entry = pl_iter.Value()
			tf_d = float64(pl_entry.Frequency())
			doc_info = index.DocumentMap[pl_entry.DocId()]

			/* Add to the numerator for each document. We'll divide later */
			partial_score = tf_d * (bm.k1 + 1)
			partial_score /= tf_d +
				bm.k1*((1.0-bm.b)+(bm.b*(float64(doc_info.TermCount)/avgDocLen)))

			partial_score *=
				(float64((bm.k2+1)*q_term_tf) / float64(bm.k2*q_term_tf))

			docScores[pl_entry.DocId()] +=
				indexer.Idf(term, index.DocumentCount) * partial_score
		}
	}

	responseSet := NewResponse()
	for id, score := range docScores {
		doc_info = index.DocumentMap[id]

		log.Debugf("Doc: %s, Score: %0.4f", doc_info.HumanId, score)
		responseSet.Append(&Result{doc_info.HumanId, score})
	}
	log.Debugf("identified result set: %v", responseSet)

	sort.Sort(responseSet)
	return responseSet
}
