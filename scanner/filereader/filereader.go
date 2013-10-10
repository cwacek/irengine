package filereader

type DocumentId int64

type Document interface {
  OrigIdent() string
  Identifier() DocumentId
  Tokens() <-chan *Token
  Len() int /* The number of tokens in this document */
  Add(*Token) /* Add a token, setting the DocId and position if necessary */
}

type FileReader interface {
  Init(string)
  ReadAll() <-chan Document
  Read() Document
  Path() string
}

