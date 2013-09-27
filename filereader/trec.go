package filereader

import log "github.com/cihub/seelog"
import "fmt"
import "os"
import "io"
import "bytes"

type TrecDocument struct {
	tokens []*Token
	id   string
}

func NewTrecDocument() (*TrecDocument) {
  doc := new(TrecDocument)
  doc.id = ""
  doc.tokens = make([]*Token, 0)
  return doc
}

func (d *TrecDocument) Tokens() <-chan *Token{

  c := make(chan *Token)
  go func(c chan *Token, tokens []*Token) {

    for _, token := range tokens {
      c <- token
    }
    close(c)
  }(c, d.tokens)

  return c
}

func (d *TrecDocument) Identifier() string {
	return d.id
}

type TrecFileReader struct {
	filename   string
	docCounter int
  scanner    Tokenizer
	documents  chan Document
}

func (fr *TrecFileReader) Init(filename string) {
	fr.docCounter = 0
	fr.filename = filename

	if file, err := os.Open(filename); err != nil {
		panic(fmt.Sprintf("Unable to open file %s", filename))
	} else {
    log.Debugf("Reading XML from %s\n", file)
    fr.scanner = BadXMLTokenizer_FromReader(file)
	}

  fr.documents = make(chan Document)
}

func (fr *TrecFileReader) DocumentsChannel() <-chan Document {
	return fr.documents
}


func (fr *TrecFileReader) read_next_doc() (Document, error) {

  var doc *TrecDocument
  var in_text, in_title bool
  var titlebuf = new(bytes.Buffer)

  for token := range fr.scanner.Tokens() {
    /*log.Debugf("Going to read another token")*/
    /*token, ok := <- tokens*/
    log.Debugf("Read token '%s'", token)

    switch {
    case token.Type == XMLStartToken && token.Text == "DOC":
      log.Debugf("Start Document")
      doc = NewTrecDocument()
    case token.Type == XMLEndToken && token.Text == "DOC":
      if doc == nil {
        panic(fmt.Sprintf("Found %s before DOC beginning", token))
      }
      log.Debugf("Return Document %v", doc)
      return doc, nil

    case token.Type == XMLStartToken && token.Text == "TEXT":
      log.Debugf("Start TEXT section")
      in_text = true
      if doc == nil {
        panic(fmt.Sprintf("Found %s before DOC beginning", token))
      }
    case token.Type == XMLEndToken && token.Text == "TEXT":
      log.Debugf("End TEXT section")
      in_text = false

      /* Read document identifiers */
    case token.Type == XMLStartToken && token.Text == "DOCNO":
      in_title = true
      titlebuf.Reset()
    case token.Type == XMLEndToken && token.Text == "DOCNO":
      doc.id = titlebuf.String()
      in_title = false

    case token.Type == TextToken || token.Type == SymbolToken:
      log.Debugf("Read token %s. Title: %v; Text: %v",token, in_title, in_text)
      switch {
      case in_title && token.Type == TextToken:
        titlebuf.WriteString(token.Text)
      case in_text:
        log.Debugf("Adding %s to document tokens. Doc is %s", token, doc)
        doc.tokens = append(doc.tokens, token)
      }
    }
  }
  log.Infof("Tokenizer returned.")
  return nil, io.EOF
}

func (fr *TrecFileReader) read_to_chan(count int) (i int) {

  for i := 0; i < count; i++ {
    log.Infof("Reading document %d", i)
		doc, ok := fr.read_next_doc()
    log.Infof("Read document. Error: %v", ok)

		if ok == io.EOF {
      break
      close(fr.documents)
		}

    log.Infof("Pushing document %s to channel %v", doc.Identifier(), fr.documents)
		fr.documents <- doc
	}
  log.Infof("Returning")
	return i
}

func (fr *TrecFileReader) Read() Document {
	go fr.read_to_chan(1)
  log.Infof("Waiting to read from channel")
	doc := <-fr.documents
  log.Infof("Read Document %s from channel", doc.Identifier())
	return doc
}

func (fr *TrecFileReader) ReadAll() <-chan Document {
  fr.scanner.Reset()
	go fr.read_to_chan(-1)
	return fr.documents
}
