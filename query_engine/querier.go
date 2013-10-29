package query_engine

import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"

type RelevanceRanker interface {
	ProcessQuery([]*filereader.Token, *indexer.SingleTermIndex) Response
}
