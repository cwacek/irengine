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
	ProcessQuery([]*filereader.Token, *indexer.SingleTermIndex) Response
}
