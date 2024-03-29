package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "math"
import "sort"
import "fmt"

type DirichletQL struct {
	mu float64
}

func init() {
	RegisterRankingEngine("LM", &DirichletQL{0})
}

func (lm *DirichletQL) ProcessPositional(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	var (
		pl                  indexer.PostingList
		partial_score, tf_d float64
		pl_entry            indexer.PostingListEntry
		term                indexer.LexiconTerm
		doc_info            *indexer.StoredDocInfo
	)

	pl = FilterPositional(query_terms, index)

	if pl == nil {
		return ErrorResponse("Could not find phrase using positional posting list")
	}

	if !force && pl.Len() < int(0.01*float64(index.DocumentCount)) {
		return ErrorResponse(fmt.Sprintf("Insufficient DF [%d/%d] for positional index",
			pl.Len(), index.DocumentCount))
	}

	avgDocLen := float64(index.TermCount()) / float64(index.DocumentCount)
	if lm.mu == 0 {
		lm.mu = math.Sqrt(avgDocLen)
	}

	//Make a fake term with the posting list
	term = &indexer.Term{"<phrase>", -1, pl}

	docScores := make(map[filereader.DocumentId]float64)

	pl = term.PostingList()
	for pl_iter := pl.Iterator(); pl_iter.Next(); {
		pl_entry = pl_iter.Value()
		tf_d = float64(pl_entry.Frequency())
		doc_info = index.DocumentMap[pl_entry.DocId()]

		log.Debugf("Obtained PL Entry %s with frequency %f", pl_entry.Serialize(), tf_d)

		partial_score = tf_d
		log.Debugf("Tf_d %f", tf_d)
		partial_score += lm.mu *
			(float64(term.Tf()) / float64(index.TermCount()))
		log.Debugf("numerator: %f", partial_score)
		partial_score /= float64(doc_info.TermCount) + lm.mu
		log.Debugf("after division: %f. Log %f",
			partial_score, math.Log(partial_score*1000.0))

		docScores[pl_entry.DocId()] += math.Log(partial_score * 1000)
	}

	responseSet := NewResponse()
	for id, score := range docScores {
		doc_info = index.DocumentMap[id]

		log.Debugf("Doc: %s, Score: %0.4f", doc_info.HumanId, score)
		responseSet.Append(&Result{doc_info.HumanId, score, ""})
	}

	sort.Sort(responseSet)
	return responseSet
}

func (lm *DirichletQL) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	if index.IsPositional() {
		return lm.ProcessPositional(query_terms, index, force)
	}

	var (
		q_term    *filereader.Token
		docScores = make(map[filereader.DocumentId]float64)
		term      indexer.LexiconTerm
		ok        bool
		pl        indexer.PostingList
		pl_iter   indexer.PostingListIterator
		pl_entry  indexer.PostingListEntry
		doc_info  *indexer.StoredDocInfo
		avgDf     float64
	)

	avgDocLen := float64(index.TermCount()) / float64(index.DocumentCount)
	if lm.mu == 0 {
		lm.mu = math.Sqrt(avgDocLen)
	}

	var partial_score, tf_d float64

	var i int
	for i, q_term = range query_terms {
		term, ok = index.Retrieve(q_term.Text)
		if !ok {
			continue
		}

		avgDf += float64(indexer.Df(term))

		log.Debugf("Calculating score for query term %d: %s ",
			i, q_term.Text)

		// Iterate over the whole posting list
		pl = term.PostingList()
		for pl_iter = pl.Iterator(); pl_iter.Next(); {
			pl_entry = pl_iter.Value()
			doc_info = index.DocumentMap[pl_entry.DocId()]

			tf_d = float64(pl_entry.Frequency())

			log.Debugf("Obtained PL Entry %v with frequency %f", pl_entry, tf_d)

			partial_score = tf_d
			log.Debugf("Tf_d %f", tf_d)
			partial_score += lm.mu *
				(float64(term.Tf()) / float64(index.TermCount()))
			log.Debugf("numerator: %f", partial_score)
			log.Debugf("TermCount:%d, DocCount: %d", doc_info.TermCount, index.DocumentCount)
			partial_score /= float64(doc_info.TermCount) + lm.mu
			log.Debugf("after division: %f. Log %f",
				partial_score, math.Log(partial_score))

			//When we take the logarithm, our numbers are so small that it ends up
			// being negative. This has a pathological result because documents with *more*
			// of the query terms end up worse off than docs with fewer because they add
			// more negative numbers. Solution: multiply by 1000 before taking the log.

			docScores[pl_entry.DocId()] += math.Log(partial_score * 1000.0)
			log.Debugf("Added %f to docScore for %s. Total: %f",
				math.Log(partial_score*1000.0), pl_entry.DocId(), docScores[pl_entry.DocId()])
		}
	}

	if !force && avgDf < float64(index.DocumentCount)*0.01 {
		return ErrorResponse(fmt.Sprintf("Avg DF %0.4f too low for index", avgDf))
	}

	responseSet := NewResponse()
	for id, score := range docScores {
		doc_info = index.DocumentMap[id]

		log.Debugf("Doc: %s, Score: %0.4f", doc_info.HumanId, score)
		responseSet.Append(&Result{doc_info.HumanId, score, ""})
	}

	sort.Sort(responseSet)
	return responseSet
}
