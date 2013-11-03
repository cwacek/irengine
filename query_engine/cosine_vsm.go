package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "math"
import "fmt"
import "sort"

func init() {
	RegisterRankingEngine("COSINE", &CosineVSM{})
}

type CosineVSM struct {
}

func (vsm *CosineVSM) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	if index.IsPositional() {
		return vsm.ProcessPositional(query_terms, index, force)
	}

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
		avgDf     float64
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

		avgDf += float64(indexer.Df(term))

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

	if !force && avgDf < float64(index.DocumentCount)*0.05 {
		return ErrorResponse(fmt.Sprintf("Avg DF %0.4f too low for index", avgDf))
	}

	log.Debugf("Computed a query weight of %0.4f", query_weight)

	/* Now we need to sum the square of the document weights for every term
	 * *in the document*. We'll use the list of documents we obtained as
	 * partial scores. */

	responseSet := NewResponse()
	var doc_weight, term_tfidf float64
	var doc_info *indexer.StoredDocInfo

	for id, numerator := range docScores {

		doc_info = index.DocumentMap[id]

		for _, term_tfidf = range doc_info.TermTfIdf {
			doc_weight += math.Pow(term_tfidf, 2.0)
		}

		log.Debugf("Document weight for %s is %0.4f", doc_info.HumanId, doc_weight)

		responseSet.Append(&Result{
			doc_info.HumanId,
			numerator / math.Sqrt(doc_weight*query_weight)})
	}

	sort.Sort(responseSet)
	log.Debugf("CosineVSM returning: %v", responseSet)
	return responseSet
}

func (cosine *CosineVSM) ProcessPositional(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	var (
		pl       indexer.PostingList
		term     indexer.LexiconTerm
		pl_entry indexer.PostingListEntry
		doc_info *indexer.StoredDocInfo
	)

	docScores := make(map[filereader.DocumentId]float64)

	pl = FilterPositional(query_terms, index)
	//pl holds hte filtered posting list

	if pl == nil {
		return ErrorResponse("Could not find phrase using positional posting list")
	}

	if !force && pl.Len() < int(0.01*float64(index.DocumentCount)) {
		return ErrorResponse(fmt.Sprintf("Insufficient DF [%d/%d] for positional index",
			pl.Len(), index.DocumentCount))
	}

	//Calculate the scores
	var partial float64
	for pl_iter := pl.Iterator(); pl_iter.Next(); {
		pl_entry = pl_iter.Value()

		/* Add to the numerator for each document. We'll divide later */
		partial = float64(pl_entry.Frequency()) *
			indexer.Idf(term, index.DocumentCount)

		log.Debugf("Computed dot-product partial: %0.4f", partial)
		docScores[pl_entry.DocId()] += partial
	}

	responseSet := NewResponse()
	var doc_weight, term_tfidf float64

	for id, numerator := range docScores {

		doc_info = index.DocumentMap[id]

		for _, term_tfidf = range doc_info.TermTfIdf {
			doc_weight += math.Pow(term_tfidf, 2.0)
		}

		log.Debugf("Document weight for %s is %0.4f", doc_info.HumanId, doc_weight)

		responseSet.Append(&Result{
			doc_info.HumanId,
			numerator / math.Sqrt(doc_weight)})
	}

	sort.Sort(responseSet)
	log.Debugf("CosineVSM returning: %v", responseSet)
	return responseSet
}

func NewCosineVSM() RelevanceRanker {
	return &CosineVSM{}
}
