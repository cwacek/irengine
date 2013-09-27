package filereader

import log "github.com/cihub/seelog"

func setupLogging(config_xml string) {
  logger, err := log.LoggerFromConfigAsFile(config_xml)

  if err  != nil {
    panic(err)
  }

  log.ReplaceLogger(logger)
}

type Document interface {
  Identifier() string
  Tokens() <-chan *Token
  Len() int /* The number of tokens in this document */
}

type FileReader interface {
  Init(string)
  ReadAll() <-chan Document
  Read() Document
}
