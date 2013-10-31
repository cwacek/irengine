package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "math"
import "sort"

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
		partial   float64
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
			partial = float64(pl_entry.Frequency()) *
				indexer.Idf(term, index.DocumentCount) *
				float64(query_tf[q_term.Text])
			log.Debugf("Computed dot-product partial: %0.4f", partial)
			docScores[pl_entry.DocId()] += partial
		}

		query_weight += math.Pow(float64(query_tf[q_term.Text])*1, 2.0)
	}

	log.Debugf("Computed a query weight of %0.4f", query_weight)

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

		log.Debugf("Document weight for %s is %0.4f", doc_info.HumanId, doc_weight)

		responseSet = append(responseSet, &Result{doc_info.HumanId,
			numerator / math.Sqrt(doc_weight*query_weight)})
	}

	sort.Sort(responseSet)
	log.Debugf("CosineVSM returning: %v", responseSet)
	return responseSet
}

func NewCosineVSM() RelevanceRanker {
	return &CosineVSM{}
}
