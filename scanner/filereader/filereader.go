package filereader

type Document interface {
  Identifier() string
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

