package indexer

import "bytes"
import "bufio"
import "math"
import "encoding/json"
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
	RandInts      []filereader.DocumentId

	basicOutput [][]byte
)

func init() {

	rand.Seed(0)
	TestDocuments = []filereader.Document{
		filters.LoadTestDocument("A02", "Since I was a young boy; I played the silver ball."),
		filters.LoadTestDocument("A03", "Since Ph.D's don't fly F-16 jets, but they might work for the CDC on the CDC-50 project"),
	}

	RandInts = make([]filereader.DocumentId, 2)
	RandInts[0] = TestDocuments[0].Identifier()
	RandInts[1] = TestDocuments[1].Identifier()

	basicOutput = [][]byte{
		[]byte(fmt.Sprintf("1. 'a' [1]: %d 4", RandInts[0])),
		[]byte(fmt.Sprintf("2. 'ball' [1]: %d 11", RandInts[0])),
		[]byte(fmt.Sprintf("3. 'boy' [1]: %d 6", RandInts[0])),
		[]byte(fmt.Sprintf("4. 'but' [1]: %d 7", RandInts[1])),
		[]byte(fmt.Sprintf("5. 'cdc' [2]: %d 13 16", RandInts[1])),
		[]byte(fmt.Sprintf("6. 'cdc50' [1]: %d 16", RandInts[1])),
		[]byte(fmt.Sprintf("7. 'dont' [1]: %d 3", RandInts[1])),
		[]byte(fmt.Sprintf("8. 'f16' [1]: %d 5", RandInts[1])),
		[]byte(fmt.Sprintf("9. 'fly' [1]: %d 4", RandInts[1])),
		[]byte(fmt.Sprintf("10. 'for' [1]: %d 11", RandInts[1])),
		[]byte(fmt.Sprintf("11. 'i' [2]: %d 2 7", RandInts[0])),
		[]byte(fmt.Sprintf("12. 'jets' [1]: %d 6", RandInts[1])),
		[]byte(fmt.Sprintf("13. 'might' [1]: %d 9", RandInts[1])),
		[]byte(fmt.Sprintf("14. 'on' [1]: %d 14", RandInts[1])),
		[]byte(fmt.Sprintf("15. 'phds' [1]: %d 2", RandInts[1])),
		[]byte(fmt.Sprintf("16. 'played' [1]: %d 8", RandInts[0])),
		[]byte(fmt.Sprintf("17. 'project' [1]: %d 17", RandInts[1])),
		[]byte(fmt.Sprintf("18. 'silver' [1]: %d 10", RandInts[0])),
		[]byte(fmt.Sprintf("19. 'since' [2]: %d 1 | %d 1", RandInts[1], RandInts[0])),
		[]byte(fmt.Sprintf("20. 'the' [3]: %d 12 15 | %d 9", RandInts[1], RandInts[0])),
		[]byte(fmt.Sprintf("21. 'they' [1]: %d 8", RandInts[1])),
		[]byte(fmt.Sprintf("22. 'was' [1]: %d 3", RandInts[0])),
		[]byte(fmt.Sprintf("23. 'work' [1]: %d 10", RandInts[1])),
		[]byte(fmt.Sprintf("24. 'young' [1]: %d 5", RandInts[0])),
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

	var index *SingleTermIndex
	var lexicon Lexicon

	lexicon = NewTrieLexicon()

	index = new(SingleTermIndex)
	index.Init(lexicon)
	rand.Seed(0)

	filterChain := filters.NewAcronymFilter()
	filterChain = filterChain.Connect(filters.NewHyphenFilter(), false)
	filterChain = filterChain.Connect(filters.NewLowerCaseFilter(), false)
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

	if term, ok := index.Retrieve("since"); !ok {
		t.Errorf("Failed to find expected term 'since' in index")
	} else {
		if idf := term.Idf(index.DocumentCount); idf != 1 {
			t.Errorf("Failed to compute IDF. Expected %0.6f. Got %0.6f",
				1, idf)
		}

		if tf := term.Tf(); tf != 2 {
			t.Errorf("Failed to compute TF. Expected %d. Got %d",
				2, tf)
		}
	}

	if term, ok := index.Retrieve("jets"); !ok {
		t.Errorf("Failed to find expected term 'since' in index")
	} else {
		expected := 1 + math.Log10(2.0/1.0)
		if idf := term.Idf(index.DocumentCount); idf != expected {
			t.Errorf("Failed to compute IDF. Expected %0.6f. Got %0.6f",
				expected, idf)
		}

		if tf := term.Tf(); tf != 1 {
			t.Errorf("Failed to compute TF. Expected %d. Got %d",
				1, tf)
		}
	}

	index.Delete()
}

func TestDocMapSerialize(t *testing.T) {
	logging.SetupTestLogging()

	var info1 = &StoredDocInfo{
		filereader.DocumentId(10),
		"Fred",
		64,
		1,
		map[string]float64{"test": 2.42},
	}
	var expected1 = `{"Id":10,"HumanId":"Fred","TermCount":64,"MaxTf":1,"TermTfIdf":{"test":2.42}}`

	var info2 = &StoredDocInfo{
		filereader.DocumentId(11),
		"James",
		64,
		1,
		make(map[string]float64),
	}

	var buf = new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	if enc.Encode(info1); strings.TrimSpace(buf.String()) != expected1 {
		t.Errorf("Expected '%s'. Got '%s'", expected1, buf.String())
	}

	var docmap = make(DocInfoMap)
	docmap[info1.Id] = info1
	docmap[info2.Id] = info2
	expectedmap := fmt.Sprintf(`[%s,{"Id":11,"HumanId":"James","TermCount":64,"MaxTf":1,"TermTfIdf":{}}]`, expected1)

	if bytes, err := json.Marshal(docmap); err != nil {
		t.Errorf("error marshalling: %v", err)
	} else if string(bytes) != expectedmap {

		t.Errorf("Expected:\n%s\n Got:\n%s", expectedmap, string(bytes))
	}

	docmap = make(DocInfoMap)
	if err := json.Unmarshal([]byte(expectedmap), &docmap); err != nil {
		t.Errorf("Error unmarshalling DocInfoMap: %s", err.Error())
	}

	if bytes, err := json.Marshal(docmap); err != nil {
		t.Errorf("error marshalling: %v", err)
	} else if string(bytes) != expectedmap {

		t.Errorf("Reserializing de-serialized docmap failed. Expected:\n%s\n Got:\n%s", expectedmap, string(bytes))
	}

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
		t.Logf("Scanned %s", scanner.Text())

		pl_entry := NewBasicEntry(0)

		parsed, e = fmt.Sscanln(scanner.Text(), &term, pl_entry)
		if parsed != 2 {
			t.Errorf("Basic: Parsed wrong number of terms (%d) from %s", parsed, scanner.Text())
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
		t.Logf("Scanned %s", scanner.Text())

		pl_entry := NewPositionalEntry(0)

		parsed, e = fmt.Sscanln(scanner.Text(), &term, pl_entry)
		if parsed != 2 {
			t.Errorf("Positional: Parsed wrong number of terms (%d) from %s", parsed, scanner.Text())
		}
		if e != nil {
			t.Error(e)
		}
		log.Flush()

		t.Logf("Scanned %s into %s", scanner.Text(), pl_entry.Serialize())
	}

}
