package indexer

import "bytes"
import "bufio"
import "strings"
import "fmt"
import "testing"
import "math/rand"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/logging"
import "github.com/cwacek/irengine/indexer/filters"
import log "github.com/cihub/seelog"

var (
   TestDocuments []filereader.Document
	RandInts []filereader.DocumentId

	basicOutput [][]byte
)

func init() {

	rand.Seed(0)
	TestDocuments = []filereader.Document{
		filters.LoadTestDocument("A02", "Since I was a young boy; I played the silver ball."),
		filters.LoadTestDocument("A03", "Since Ph.D's don't fly F-16 jets, but they might work for the CDC on the CDC-50 project"),
	}

	rand.Seed(0)
	RandInts = make([]filereader.DocumentId, 20)
	for x := 0; x < 20; x++ {
		RandInts[x] = filereader.DocumentId(rand.Int63())
    log.Infof("RandInt[%d] = %d", x, RandInts[x])
	}

  basicOutput = [][]byte{
		[]byte(fmt.Sprintf("1. 'a' [1]: %d 4", RandInts[0])),
		[]byte(fmt.Sprintf("2. 'ball' [1]: %d 11", RandInts[0])),
		[]byte(fmt.Sprintf("3. 'boy' [1]: %d 6", RandInts[0])),
		[]byte(fmt.Sprintf("4. 'but' [1]: %d 7", RandInts[1])),
		[]byte(fmt.Sprintf("5. 'cdc' [2]: %d 13,16", RandInts[1])),
		[]byte(fmt.Sprintf("6. 'cdc50' [1]: %d 16", RandInts[1])),
		[]byte(fmt.Sprintf("7. 'dont' [1]: %d 3", RandInts[1])),
		[]byte(fmt.Sprintf("8. 'f16' [1]: %d 5", RandInts[1])),
		[]byte(fmt.Sprintf("9. 'fly' [1]: %d 4",RandInts[1])),
		[]byte(fmt.Sprintf("10. 'for' [1]: %d 11",RandInts[1])),
		[]byte(fmt.Sprintf("11. 'i' [2]: %d 2,7",RandInts[0])),
		[]byte(fmt.Sprintf("12. 'jets' [1]: %d 6",RandInts[1])),
		[]byte(fmt.Sprintf("13. 'might' [1]: %d 9",RandInts[1])),
		[]byte(fmt.Sprintf("14. 'on' [1]: %d 14",RandInts[1])),
		[]byte(fmt.Sprintf("15. 'phds' [1]: %d 2",RandInts[1])),
		[]byte(fmt.Sprintf("16. 'played' [1]: %d 8",RandInts[0])),
		[]byte(fmt.Sprintf("17. 'project' [1]: %d 17",RandInts[1])),
		[]byte(fmt.Sprintf("18. 'silver' [1]: %d 10",RandInts[0])),
		[]byte(fmt.Sprintf("19. 'since' [2]: %d 1 | %d 1",RandInts[0],RandInts[1])),
		[]byte(fmt.Sprintf("20. 'the' [3]: %d 9 | %d 12,15",RandInts[0],RandInts[1])),
		[]byte(fmt.Sprintf("21. 'they' [1]: %d 8",RandInts[1])),
		[]byte(fmt.Sprintf("22. 'was' [1]: %d 3",RandInts[0])),
		[]byte(fmt.Sprintf("23. 'work' [1]: %d 10",RandInts[1])),
		[]byte(fmt.Sprintf("24. 'young' [1]: %d 5",RandInts[0])),
	}
}

func TestSingleTermIndex(t *testing.T) {
	logging.SetupTestLogging()

	defer func() {
		if x := recover(); x != nil {
			log.Criticalf("Error: %v", x)
			log.Flush()
		}
	}()

	var index Indexer
	var lexicon Lexicon

	lexicon = NewTrieLexicon()

	index = new(SingleTermIndex)
	index.Init(lexicon)
  rand.Seed(0)

	filterChain := filters.NewAcronymFilter("acronyms")
	filterChain = filterChain.Connect(filters.NewHyphenFilter("hyphens"), false)
	filterChain = filterChain.Connect(filters.NewLowerCaseFilter("lower"), false)
	index.AddFilter(filterChain)

	for _, document := range TestDocuments {
		log.Debugf("Inserting %s", document)
		index.Insert(document)
		log.Debugf("Finished inserting %s", document)
	}

	log.Infof("Inserted all documents into %s", index.String())

	output := new(bytes.Buffer)
	index.PrintLexicon(output)

	for i, expected := range basicOutput {
		if line, err := output.ReadBytes('\n'); err != nil {
			t.Errorf("Error reading lexicon output at line %d. Expected '%s'", i+1, expected)
			break
		} else {

			trimmed := bytes.TrimSpace(line)

			if !bytes.Equal(trimmed, expected) {
				t.Errorf("Mismatched lexicon output at line %d: Expected '%s' (len: %d). Got '%s' (len: %d)",
					i+1, expected, len(expected), line, len(line))
			}
		}
	}

	index.Delete()
}

func TestCanLoad(t *testing.T) {
  logging.SetupTestLogging()

  var term string
  var e error
  var parsed int

  test := `
  ancer 2026317775 34
  cancer 1860504235 1
  cancer 1714195348 1
  cancer 867431364 21
  cancer 700694891 5
  cancer 550589489 2
  cancer 967637766 1
  cancer 973550529 3
  cancer 1616807793 1
  cancer 1625693750 1
  cancer 1703402506 1
  cancer 1985643851 5
  erstwhile 884772248 1
  corrosion 712138534 1
  corrosion 562925694 1
  corrosion 227215139 6
  corrosion 1965206303 1`

  test2 := `
  ancer 2026317775 12 23 48 
  cancer 1860504235 1 2 3 
  cancer 1714195348 1
  cancer 867431364 21
  cancer 700694891 5
  cancer 550589489 2 7 9 0
  cancer 967637766 1
  cancer 973550529 3
  cancer 1616807793 1
  cancer 1625693750 1
  cancer 1703402506 1
  cancer 1985643851 5
  erstwhile 884772248 1
  corrosion 712138534 1
  corrosion 562925694 1
  corrosion 227215139 6
  corrosion 1965206303 1`

  scanner := bufio.NewScanner(strings.NewReader(test))
  scanner.Split(bufio.ScanLines)

  for scanner.Scan() {
    if len(scanner.Text()) == 0 {
      continue
    }
    log.Infof("Scanned %s", scanner.Text())

    pl_entry := NewBasicEntry(0)

    parsed, e = fmt.Sscanln(scanner.Text(), &term, pl_entry)
    if parsed != 2 {
      t.Errorf("Basic: Parsed wrong number of terms (%d) from %s",parsed, scanner.Text())
    }
    if e != nil {
      t.Error(e)
    }
  }


  scanner = bufio.NewScanner(strings.NewReader(test2))
  scanner.Split(bufio.ScanLines)

  for scanner.Scan() {
    if len(scanner.Text()) == 0 {
      continue
    }
    log.Infof("Scanned %s", scanner.Text())

    pl_entry := NewPositionalEntry(0)

    parsed, e = fmt.Sscanln(scanner.Text(), &term, pl_entry)
    if parsed != 2 {
      t.Errorf("Positional: Parsed wrong number of terms (%d) from %s",parsed, scanner.Text())
    }
    if e != nil {
      t.Error(e)
    }
    log.Flush()

    log.Infof("Scanned %s into %s", scanner.Text(), pl_entry.Serialize())
  }

}
