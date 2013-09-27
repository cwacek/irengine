package main

import log "github.com/cihub/seelog"
import "fmt"

var appConfig = `
  <seelog type="sync" minlevel='%s'>
  <outputs formatid="scanner">
    <console />
  </outputs>
  <formats>
  <format id="scanner" format="scanner: [%%LEV] %%Msg%%n" />
  </formats>
  </seelog>
`

var config string

func SetupLogging(verbosity int) {

  switch verbosity {
  case 0:
    fallthrough
  case 1:
    config = fmt.Sprintf(appConfig, "warn")
  case 2:
    config = fmt.Sprintf(appConfig, "info")
  case 3:
    fallthrough
  default:
    config = fmt.Sprintf(appConfig, "trace")
  }

	logger, err := log.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Println(err)
		return
	}

	log.ReplaceLogger(logger)
}
