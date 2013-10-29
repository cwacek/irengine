package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "math"

type CosineVSM struct {
}

func (vsm *CosineVSM) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex) Response {

	var (
		q_term    *filereader.Token
		query_tf  = make(map[string]int)
		docScores = make(map[filereader.DocumentId]float64)
		term      indexer.LexiconTerm
		ok        bool
		pl        indexer.PostingList
		pl_iter   indexer.PostingListIterator
		pl_entry  indexer.PostingListEntry
	)

	for _, q_term = range query_terms {
		query_tf[q_term.Text]++
		log.Infof("Processing token %s. Have %d", q_term, query_tf[q_term.Text])
	}

	query_weight := 0.0
	/* For each term in the query */
	for _, q_term = range query_terms {
		term, ok = index.Retrieve(q_term.Text)
		if !ok {
			continue
		}

		pl = term.PostingList()
		for pl_iter = pl.Iterator(); pl_iter.Next(); {
			pl_entry = pl_iter.Value()
			/* Add to the numerator for each document. We'll divide later */
			docScores[pl_entry.DocId()] += float64(pl_entry.Frequency()) *
				term.Idf(index.DocumentCount) *
				float64(query_tf[q_term.Text])
		}

		query_weight += math.Pow(float64(query_tf[q_term.Text])*1, 2.0)
	}

	/* Now we need to sum the square of the document weights for every term
	 * *in the document*. We'll use the list of documents we obtained as
	 * partial scores. */

	responseSet := make(Response, 0)
	var doc_weight, term_tfidf float64
	var doc_info *indexer.StoredDocInfo

	for id, numerator := range docScores {

		doc_info = index.DocumentMap[id]

		for _, term_tfidf = range doc_info.TermTfIdf {
			doc_weight += math.Pow(term_tfidf, 2.0)
		}

		responseSet = append(responseSet, Result{doc_info.HumanId,
			numerator / math.Sqrt(doc_weight*query_weight)})
	}

	return responseSet
}

func NewCosineVSM() RelevanceRanker {
	return &CosineVSM{}
}
