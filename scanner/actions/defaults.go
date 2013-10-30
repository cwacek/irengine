package actions

import "flag"
import "os"
import "path/filepath"
import "regexp"
import "fmt"
import log "github.com/cihub/seelog"
import filereader "github.com/cwacek/irengine/scanner/filereader"

type Args struct {
	verbosity *int
}

func (a *Args) AddDefaultArgs(fs *flag.FlagSet) {

	a.verbosity = fs.Int("v", 0, "Be verbose [1, 2, 3]")
}

type DocWalker struct {
	output       chan filereader.Document
	workers      chan string
	worker_count int
	filepattern  string
}

func (d *DocWalker) WalkDocuments(docroot, pattern string,
	out chan filereader.Document) {

	d.output = out
	d.workers = make(chan string)
	d.worker_count = 0
	d.filepattern = pattern

	log.Infof("Reading documents matching %s from: %s", pattern, docroot)
	filepath.Walk(docroot, d.read_file)
	if d.worker_count == 0 {
		close(d.output)
	}

	go d.signal_when_done()
}

func (d *DocWalker) signal_when_done() {
	for {
		select {
		case file := <-d.workers:
			d.worker_count -= 1
			log.Infof("Worker for %s done. Waiting for %d workers.", file, d.worker_count)
			if d.worker_count <= 0 {
				fmt.Println("Finished reading documents")
				close(d.output)
				return
			}
		}
	}
}

func (d *DocWalker) read_file(path string, info os.FileInfo, err error) error {

	if err != nil {
		log.Criticalf("Error walking documents at %s: %v", path, err)
		return nil
	}

	if info.Mode().IsRegular() {
		file := filepath.Base(path)

		log.Debugf("Trying file %s", file)

		matched, err := regexp.MatchString(d.filepattern, file)
		log.Debugf("File match: %v, error: %v", matched, err)
		if matched && err == nil {

			fr := new(filereader.TrecFileReader)
			fr.Init(path)

			go func() {
				for doc := range fr.ReadAll() {
					d.output <- doc
				}
				d.workers <- fr.Path()
				return
			}()

			d.worker_count += 1
			/*log.Errorf("Now have %d workers", d.worker_count)*/
		}
	}
	return nil
}

var appConfig = `
  <seelog minlevel='%s'>
  <outputs formatid="scanner">
    <filter levels="critical,error,warn,info">
      <console formatid="scanner" />
    </filter>
    <filter levels="debug,trace">
      <console formatid="debug" />
    </filter>
  </outputs>
  <formats>
  <format id="scanner" format="[%%Time]:%%LEVEL:: %%Msg%%n" />
  <format id="debug" format="[%%Time]:%%LEVEL:%%Func:: %%Msg%%n" />
  </formats>
`

var config string

func SetupLogging(verbosity int) {

	switch verbosity {
	case 0:
		fallthrough
	case 1:
		config = fmt.Sprintf(appConfig, "warn")
		fmt.Printf("Configured logging at 'warn'\n")
	case 2:
		config = fmt.Sprintf(appConfig, "info")
		fmt.Printf("Configured logging at 'info'\n")
	case 3:
		config = fmt.Sprintf(appConfig, "debug")
		fmt.Printf("Configured logging at 'debug'\n")
	default:
		config = fmt.Sprintf(appConfig, "trace")
		fmt.Printf("Configured logging at 'trace'\n")
	}

	logger, err := log.LoggerFromConfigAsBytes([]byte(config))

	if err != nil {
		fmt.Println(err)
		return
	}

	log.ReplaceLogger(logger)
}
