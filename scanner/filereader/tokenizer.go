package filereader

import "text/scanner"
import "io"
import log "github.com/cihub/seelog"
import "fmt"
import "bytes"
import "unicode"

type TokenType int

const (
    NullToken = 0
    TextToken TokenType = iota
    XMLStartToken
    XMLEndToken
    SymbolToken
)

type Token struct {
    Text string
    Type TokenType
    DocId string
    Position int
    Final bool
}

func (t *Token) Clone() *Token {
  newtok := NewToken(t.Text, t.Type)
  newtok.DocId = t.DocId
  newtok.Position = t.Position
  newtok.Final = t.Final
  return newtok
}

func (t *Token) Equal(other *Token) (equal bool) {
  equal = true

  equal = t.Eql(other)

  if t.Position != other.Position {
  equal = false
}

  return
}


func (t *Token) Eql(other *Token) (equal bool) {
  equal = true

  if t.Text != other.Text {
    equal = false
  }

  if t.Type != other.Type {
    equal = false
  }

  return
}

func NewToken(text string, ttype TokenType) (*Token) {
  t := new(Token)
  t.Text = text
  t.Type = ttype
  t.Final = false

  return t
}

type Tokenizer interface {
    Next() (*Token, error)
    Tokens() <-chan *Token
    Reset()
}

func (t TokenType) String() string {
    switch t {
        case TextToken: return "TEXT"
        case XMLStartToken: return "XMLSTART"
        case XMLEndToken: return "XMLEND"
        case SymbolToken: return "SYMBOL"
        default: return "UNKNOWN"
    }
}

func (t Token) String() string {
    return fmt.Sprintf("%s [%s@%d]", t.Text, t.Type, t.Position)
}

type BadXMLTokenizer struct{
    tok_start, tok_end int
    scanner  *scanner.Scanner
    rd io.ReadSeeker
}

func BadXMLTokenizer_FromReader(rd io.ReadSeeker) (Tokenizer){
    t := new(BadXMLTokenizer)
    t.rd = rd
    t.scanner = new(scanner.Scanner).Init(rd)
    t.scanner.Whitespace = 0
    t.scanner.Error = func(s *scanner.Scanner, msg string) { panic(msg)}
    t.scanner.Mode = scanner.ScanStrings
    return t
}

func (tz *BadXMLTokenizer) Reset() {
    _, err := tz.rd.Seek(0, 0)
    if err != nil {
        panic(err)
    }
    tz.scanner = new(scanner.Scanner).Init(tz.rd)
    tz.scanner.Whitespace = 0
    tz.scanner.Error = func(s *scanner.Scanner, msg string) { panic(msg)}
    tz.scanner.Mode = scanner.ScanStrings
}

var alnum = []*unicode.RangeTable{unicode.Digit, unicode.Letter,
unicode.Dash, unicode.Hyphen}

var symbols = []*unicode.RangeTable{unicode.Symbol,
unicode.Punct}

func (tz *BadXMLTokenizer) Next() (*Token, error) {

    for {
        tok := tz.scanner.Peek()
        log.Debugf("Scanner found: %v", tok)

        if tok == scanner.EOF {
            return nil, io.EOF
        }

        switch  {
        case unicode.IsPrint(tok) == false:
            log.Debug("Skipping unprintable character")
            fallthrough
        case unicode.IsSpace(tok):
            tz.scanner.Scan()
            continue

        case tok == '<':
            log.Debugf("parsing XML")
            token, ok := parseXML(tz.scanner)
            if ok {
                log.Debugf("Returning XML Token: %s", token)
                return token, nil
            }

        case tok == '&':
            log.Debugf("parsing HTML")
            token, ok := parseHTMLEntity(tz.scanner)
            if ok {
                return token, nil
            }

        case unicode.Is(unicode.Punct, tok):
            log.Debugf("Ignoring punctuation: %v", tok)
            tok = tz.scanner.Scan()

        default:
            /* Catch special things in words */
            log.Debugf("Found '%s' . Parsing Text", string(tok))
            token, ok := tz.parseCompound()
            if ok {
                return token, nil
            } else {
                tz.scanner.Scan()
            }
        }
    }
}

func (t *BadXMLTokenizer) parseCompound() (*Token, bool) {
    var entity = new(bytes.Buffer)

    for {
      next := t.scanner.Peek()
      log.Debugf("Next is '%v'. Text entity is %s",
      next, entity.String())

      switch {

      case next == '&':
        log.Debugf("parsing HTML")
        token, _ := parseHTMLEntity(t.scanner)
        entity.WriteString(token.Text)

      case next == '<':
        if entity.Len() > 0 {
          tok := NewToken(entity.String(), TextToken)
          return tok, true
        } else {
          return nil, false
        }

      case unicode.IsOneOf(alnum, next):
        t.scanner.Scan()
        entity.WriteString(t.scanner.TokenText())

      case unicode.IsOneOf(symbols, next):
        t.scanner.Scan()
        part2, ok := t.parseCompound()

        switch {

        case ok && next == '\'':
          if ok {
             entity.WriteString(part2.Text)
          }

        case ok:
          entity.WriteRune(next)
          entity.WriteString(part2.Text)

        case unicode.Is(unicode.Sc, next): //currency
          entity.WriteRune(next)
        }

      default:
        if entity.Len() > 0 {
          tok := NewToken(entity.String(), TextToken)
          return tok, true
        } else {
          return nil, false
        }
      }
    }
}

func decodeEntity(entity string) (string, bool) {

    switch entity {
    case "&hyph;":
        return "-", true
    case "&blank;":
        return "", false
    case "&lt;":
        return "<", true
    case "&gt;":
        return ">", true
    default:
        log.Debugf("Invalid character escape sequence: %s", entity)
        return "", false
    }
}

func parseHTMLEntity(sc *scanner.Scanner) (*Token, bool) {

    var entity = new(bytes.Buffer)
    log.Debugf("ParseHTML. Starting with '%s'", entity.String())

    for {
        tok := sc.Scan()

        switch {
        case unicode.IsSpace(tok):
            if entity.Len() > 1 {
                token := NewToken(entity.String(), TextToken)
                log.Debugf("ParseHTML. Returning non-HTML '%s'",
            token.Text)
                return token, false
            } else {
                return nil, false
            }

        case tok == ';':
            entity.WriteRune(tok)
            if decoded, ok := decodeEntity(entity.String()); ok {
                token := NewToken(decoded, TextToken)
                log.Debugf("ParseHTML. Returning HTML '%s'",
                token.Text)
                return token, true
            }
        default:
            entity.WriteString(sc.TokenText())
        }
    }

}

func parseXML(sc *scanner.Scanner) (*Token, bool) {

    var entity = new(bytes.Buffer)
    token := new(Token)

    // Skip the '<'
    sc.Scan()

    switch sc.Peek() {
    case '/':
        token.Type = XMLEndToken
        sc.Next()
    case '!':
        log.Debugf("parseXML skipping comment")
        next := sc.Next()
        for next != '>' {
            next = sc.Next()
        }
        return nil, false
    default:
        token.Type = XMLStartToken
    }

    log.Debugf("parseXML creating %s element", token.Type )

    for {
        tok := sc.Scan()
        log.Debugf("parseXML found %s. Token is %v. Entity is: '%s'",
        sc.TokenText(),
        tok,
        entity.String())

        switch {
        case tok == '>':
            token.Text = entity.String()
            return token, true

        case unicode.IsSpace(tok):
            return nil, false

        default:
            log.Debugf("parseXML appending %s to string",
            sc.TokenText())
            entity.WriteString(sc.TokenText())

        }
    }
}

func (tz *BadXMLTokenizer) Tokens() (<- chan *Token) {

    token_channel := make(chan *Token)
    log.Debugf("Created channel %v as part of Tokens(), with" +
              " Scanner = %v", token_channel, tz)

    go func(ret chan *Token, tz *BadXMLTokenizer) {
        for {
            log.Debugf("Scanner calling Next()")
            tok, err := tz.Next()
            log.Debugf("scanner.Next() returned %s, %v", tok, err)
            switch err {
            case nil:
                log.Debugf("Pushing %s into token channel %v",
                tok, ret)
                ret <- tok
            case io.EOF:
                log.Debugf("received EOF, closing channel")
                close(ret)
                log.Debugf("Closed.")
                log.Flush()
                return
                panic("I should have exited the goroutine but " +
                "didn't")
            }
        }
    }(token_channel, tz)

    return token_channel
}
