package query_engine

import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/scanner/filereader"

type CosineVSM struct {
}

func (vsm *CosineVSM) ProcessQuery(
	tokens []*filereader.Token,
	index *indexer.SingleTermIndex) Response {

	var token *filereader.Token

	for _, token = range tokens {
		log.Infof("Processing token %s", token)
	}

	return Response{Result{"hello", 0.2}}
}

func NewCosineVSM() RelevanceRanker {
	return &CosineVSM{}
}
