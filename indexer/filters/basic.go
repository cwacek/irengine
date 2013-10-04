package filters

import "strings"
import log "github.com/cihub/seelog"
import "bytes"
import "github.com/cwacek/irengine/scanner/filereader"

func CombineTokens(
  tokens []*filereader.Token,
	resultType filereader.TokenType) *filereader.Token {

	log.Debugf("Combining tokens %v", tokens)
	combinedText := new(bytes.Buffer)

	for _, tok := range tokens {
		combinedText.WriteString(tok.Text)
	}
	newtok := new(filereader.Token)
	newtok.Text = combinedText.String()
	newtok.Position = tokens[0].Position
	newtok.Final = false
  newtok.Type = resultType
	newtok.DocId = tokens[0].DocId

	return newtok
}

type LowerCaseFilter struct {
	FilterPlumbing
}

func NewLowerCaseFilter(id string) Filter {
	f := new(LowerCaseFilter)
	f.Id = id
	f.self = f
	return f
}

type NullFilter struct {
	FilterPlumbing
}

func NewNullFilter(id string) Filter {
	f := new(NullFilter)
	f.Id = id
	f.self = f
	return f
}

func (f *LowerCaseFilter) IsDestructive() bool {
	return true
}

func (f *LowerCaseFilter) Apply(tok *filereader.Token) ([]*filereader.Token){
  res := make([]*filereader.Token, 0, 1)

  converted := tok.Clone()
  converted.Text = strings.ToLower(tok.Text)

  res = append(res, converted)
  return res
}

func (f *NullFilter)  Apply(tok *filereader.Token) ([]*filereader.Token){
  res := make([]*filereader.Token, 0, 1)

  return res
}
