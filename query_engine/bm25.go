package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"
import "sort"
import "math"
import "fmt"

type BM25 struct {
	k1 float64
	k2 int
	b  float64
}

func init() {
	RegisterRankingEngine("BM25", &BM25{1.2, 1, 0.75})
}

func FilterPositional(query_terms []*filereader.Token,
	index *indexer.SingleTermIndex) indexer.PostingList {

	var (
		term indexer.LexiconTerm
		pl   indexer.PostingList
		ok   bool
	)

	// We're going to load the posting list for the first
	// term, then filter it against the posting list for the
	// second term, then the third. Essentially a reduction.
	// Then we'll calculate the result over the frequencies
	// of the "Query Posting List"

	within := 1

	for _, q_term := range query_terms {
		term, ok = index.Retrieve(q_term.Text)

		switch {
		case pl != nil && ok:
			log.Debugf("Filtering by PostingList for '%s': %s", term.Text(), term.PostingList())

			pl = pl.FilterSequential(term.PostingList(), within)

			log.Debugf("After filtering within %d positions, have %s", within, pl.String())

			within = 1

		case pl != nil && !ok:
			within++
			log.Debugf("Couldn't find %s in index. Looking past it")

		case pl == nil && !ok:
			// Don't increment, but continue

		default:
			pl = term.PostingList()
			log.Debugf("Postinglist for first term: %s", pl.String())
		}

	}

	return pl
}

func (bm *BM25) ProcessPositional(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	var (
		pl                  indexer.PostingList
		term                indexer.LexiconTerm
		partial_score, tf_d float64
		pl_entry            indexer.PostingListEntry
		doc_info            *indexer.StoredDocInfo
	)

	avgDocLen := float64(index.TermCount()) / float64(index.DocumentCount)

	pl = FilterPositional(query_terms, index)

	if pl == nil {
		return ErrorResponse("Could not find phrase using positional posting list")
	}

	log.Debugf("Filtered posting list. Result: %s", pl.String())

	q_term_tf := 1
	docScores := make(map[filereader.DocumentId]float64)

	//pl holds hte filtered posting list
	if !force && pl.Len() < int(0.01*float64(index.DocumentCount)) {
		return ErrorResponse(fmt.Sprintf("Insufficient DF [%d/%d] for positional index",
			pl.Len(), index.DocumentCount))
	}
	term = &indexer.Term{"<phrase>", -1, pl}

	for pl_iter := pl.Iterator(); pl_iter.Next(); {
		pl_entry = pl_iter.Value()
		tf_d = float64(1 + math.Log(float64(pl_entry.Frequency())))
		doc_info = index.DocumentMap[pl_entry.DocId()]

		log.Debugf("Obtained PL Entry %s with frequency %f", pl_entry.Serialize(), tf_d)
		/* Add to the numerator for each document. We'll divide later */
		partial_score = tf_d * (bm.k1 + 1)
		partial_score /= tf_d +
			bm.k1*((1.0-bm.b)+(bm.b*(float64(doc_info.TermCount)/avgDocLen)))

		log.Debugf("Doc TermCount: %d, avgDocLen: %f", doc_info.TermCount, avgDocLen)

		partial_score *=
			(float64((bm.k2+1)*q_term_tf) / float64(bm.k2*q_term_tf))

		docScores[pl_entry.DocId()] +=
			indexer.Idf(term, index.DocumentCount) * partial_score
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

func (bm *BM25) ProcessQuery(
	query_terms []*filereader.Token,
	index *indexer.SingleTermIndex,
	force bool,
) *Response {

	if index.IsPositional() {
		return bm.ProcessPositional(query_terms, index, force)
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
		doc_info  *indexer.StoredDocInfo
	)

	for _, q_term = range query_terms {
		query_tf[q_term.Text]++
		log.Infof("Processing token %s. Have %d", q_term, query_tf[q_term.Text])
	}

	var partial_score, tf_d float64
	avgDocLen := float64(index.TermCount()) / float64(index.DocumentCount)
	var avgDf float64

	/* For each term in the query */
	var i int
	for i, q_term = range query_terms {
		q_term_tf := query_tf[q_term.Text]
		term, ok = index.Retrieve(q_term.Text)
		if !ok {
			continue
		}

		avgDf += float64(indexer.Df(term))

		log.Debugf("Calculating score for query term %d: %s [%d]",
			i, q_term.Text, q_term_tf)

		// Iterate over the whole posting list
		pl = term.PostingList()
		for pl_iter = pl.Iterator(); pl_iter.Next(); {
			pl_entry = pl_iter.Value()
			tf_d = float64(1 + math.Log(float64(pl_entry.Frequency())))
			doc_info = index.DocumentMap[pl_entry.DocId()]

			log.Debugf("Obtained PL Entry %v with frequency %f", pl_entry, tf_d)
			/* Add to the numerator for each document. We'll divide later */
			partial_score = tf_d * (bm.k1 + 1)
			partial_score /= tf_d +
				bm.k1*((1.0-bm.b)+(bm.b*(float64(doc_info.TermCount)/avgDocLen)))

			log.Debugf("Doc TermCount: %d, avgDocLen: %f", doc_info.TermCount, avgDocLen)

			partial_score *=
				(float64((bm.k2+1)*q_term_tf) / float64(bm.k2*q_term_tf))

			docScores[pl_entry.DocId()] +=
				indexer.Idf(term, index.DocumentCount) * partial_score
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
