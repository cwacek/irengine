package filters

import "strings"
import log "github.com/cihub/seelog"
import "bytes"
import "strconv"
import "regexp"
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
  f.ignoresFinal = true
	return f
}

type DigitsFilter struct {
  FilterPlumbing
}

var DigitsRegex = regexp.MustCompile(`^((?:\d+,)*\d+)(?:\.(\d+))?$`)

func NewDigitsFilter(id string) Filter {
  f := new(DigitsFilter)
  f.Id = id
  f.self = f
  return f
}

func (f *DigitsFilter) Apply(tok *filereader.Token) ([]*filereader.Token) {
  results := make([]*filereader.Token, 0, 1)

  if m := DigitsRegex.FindStringSubmatch(tok.Text); m != nil {

    var repr = new(bytes.Buffer)

    thousands := strings.Split(m[1], ",")
    for i, entry := range thousands {
      if i > 0 && len(entry) != 3 {
        //This isn't a number.
        goto NoDigit
      }
      repr.WriteString(entry)
    }

    if len(m) > 2 {
      if decimal, err := strconv.Atoi(m[2]); err == nil {
        if decimal > 0 {
          repr.WriteString("." + m[2])
        }
      }
    }

    if repr.String() != tok.Text {
      // Only create a new one if we changed something
      results = append(results, CloneWithText(tok, repr.String()))
      return results
    }
  }

NoDigit:
  results = append(results,tok)

  return results
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
  res := make([]*filereader.Token, 1)

  converted := tok.Clone()
  converted.Text = strings.ToLower(tok.Text)

  res[0] = converted
  return res
}

func (f *NullFilter)  Apply(tok *filereader.Token) ([]*filereader.Token){
  res := make([]*filereader.Token, 1, 1)
  res[0] = tok

  return res
}
