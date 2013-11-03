package query_engine

import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"

var RankingEngines map[string]RelevanceRanker

func RegisterRankingEngine(name string, r RelevanceRanker) {
	if RankingEngines == nil {
		RankingEngines = make(map[string]RelevanceRanker)
	}
	RankingEngines[name] = r
}

type RelevanceRanker interface {
	// Return a response for a query on a given index. If force is true, return the
	// resultset whether or not the DF is deemed high enough.
	ProcessQuery([]*filereader.Token, *indexer.SingleTermIndex, bool) *Response

	// Same as process query, but do it for a positional index .
	ProcessPositional([]*filereader.Token, *indexer.SingleTermIndex, bool) *Response
}
