package filters

import "testing"
import "github.com/cwacek/irengine/scanner/filereader"
import "github.com/cwacek/irengine/logging"
import "strings"
import log "github.com/cihub/seelog"

type TestCase struct {
	FilterFunc func(string) Filter
	Input      string
	Expected   []*filereader.Token
}

func AddTestCase(id string, t TestCase) {
	TestCases[id] = t
}

var (
	TestCases = map[string]TestCase{
		"acronyms": TestCase{
			NewAcronymFilter,
			`Ph.D. Ph.D Phd PhD U.S.A USA M.S. M.S MS `,
			[]*filereader.Token{
				&filereader.Token{Text: "phd", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "phd", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "Phd", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "PhD", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "usa", Type: filereader.TextToken, Position: 5},
				&filereader.Token{Text: "USA", Type: filereader.TextToken, Position: 6},
				&filereader.Token{Text: "ms", Type: filereader.TextToken, Position: 7},
				&filereader.Token{Text: "ms", Type: filereader.TextToken, Position: 8},
				&filereader.Token{Text: "MS", Type: filereader.TextToken, Position: 9},
			},
		},
		"hyphens": TestCase{
			NewHyphenFilter,
			`CDC-50 F-16 1-hour part-of-speech pre-rebellion 141-19`,
			[]*filereader.Token{
				&filereader.Token{Text: "CDC50", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "CDC", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "F16", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "1hour", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "hour", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "part", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "of", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "speech", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "partofspeech", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "prerebellion", Type: filereader.TextToken, Position: 5},
				&filereader.Token{Text: "rebellion", Type: filereader.TextToken, Position: 5},
				&filereader.Token{Text: "141-19", Type: filereader.TextToken, Position: 5},
			},
		},
		"dates": TestCase{
			NewDateFilter,
			`10/3/2013 10-3-2013 10-03-2013 9-3-2013 9-31-13
      6/31/13 06/31/2013 10-03-95 10-03-203 January 23rd 2013 January 1st Jan 2011`,
			[]*filereader.Token{
				&filereader.Token{Text: "10_03_2013", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "10_03_2013", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "10_03_2013", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "09_03_2013", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "09_31_2013", Type: filereader.TextToken, Position: 5},
				&filereader.Token{Text: "06_31_2013", Type: filereader.TextToken, Position: 6},
				&filereader.Token{Text: "06_31_2013", Type: filereader.TextToken, Position: 7},
				&filereader.Token{Text: "10_03_1995", Type: filereader.TextToken, Position: 8},
				&filereader.Token{Text: "10-03-203", Type: filereader.TextToken, Position: 9},
				&filereader.Token{Text: "January", Type: filereader.TextToken, Position: 10},
				&filereader.Token{Text: "2013", Type: filereader.TextToken, Position: 12},
				&filereader.Token{Text: "01_23_2013", Type: filereader.TextToken, Position: 12},
				&filereader.Token{Text: "January", Type: filereader.TextToken, Position: 13},
				&filereader.Token{Text: "01_01_0000", Type: filereader.TextToken, Position: 14},
				&filereader.Token{Text: "January", Type: filereader.TextToken, Position: 15},
				&filereader.Token{Text: "2011", Type: filereader.TextToken, Position: 15},
				&filereader.Token{Text: "01_00_2011", Type: filereader.TextToken, Position: 15},
			},
		},
		"digits": TestCase{
			NewDigitsFilter,
			`10,0002,10 10,000,000 1000 1000000, 1.242 12.00 10-2`,
			[]*filereader.Token{
				&filereader.Token{Text: "10,0002,10", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "10000000", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "1000", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "1000000", Type: filereader.TextToken, Position: 4},
				&filereader.Token{Text: "1.242", Type: filereader.TextToken, Position: 5},
				&filereader.Token{Text: "12", Type: filereader.TextToken, Position: 6},
				&filereader.Token{Text: "10-2", Type: filereader.TextToken, Position: 7},
			},
		},
		"filenames": TestCase{
			NewFilenameFilter,
			`test.jpg test.pdf test.html`,
			[]*filereader.Token{
				&filereader.Token{Text: "test", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "test.jpg", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "test", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "test.pdf", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "test", Type: filereader.TextToken, Position: 3},
				&filereader.Token{Text: "test.html", Type: filereader.TextToken, Position: 3},
			},
		},
		"email": TestCase{
			NewNullFilter,
			`cwacek@gmail.com cwacek.blah@gmail.com jim`,
			[]*filereader.Token{
				&filereader.Token{Text: "cwacek@gmail.com", Type: filereader.TextToken, Position: 1},
				&filereader.Token{Text: "cwacek.blah@gmail.com", Type: filereader.TextToken, Position: 2},
				&filereader.Token{Text: "jim", Type: filereader.TextToken, Position: 3},
			},
		},
	}

	chained_order = []string{
		"digits",
		"dates",
		"hyphens",
		"acronyms",
		"filenames",
		"email",
	}
)

func TestChained(t *testing.T) {
	logging.SetupTestLogging()

	var expected = make([]*filereader.Token, 0, len(chained_order))
	var inputs = make([]string, 0, len(chained_order))
	var filters = make([]Filter, 0, len(chained_order))
	var testcaseTokenStart = 0

	for i, testname := range chained_order {
		testcase := TestCases[testname]

		for _, exp_token := range testcase.Expected {
			exp_token.Position += testcaseTokenStart
			expected = append(expected, exp_token)
		}
		testcaseTokenStart += len(testcase.Expected) + 1

		inputs = append(inputs, testcase.Input)

		f := testcase.FilterFunc(testname)
		if i > 0 {
			f.Follow(filters[len(filters)-1], false)
			log.Debugf("Chained %v after %v", f, filters[len(filters)-1])
		}

		filters = append(filters, f)
	}
	log.Debugf("Expected output tokens: %s", expected)

	input := NewFilterPipe("test")
	log.Debugf("Created input pipe %v", input)
	filters[0].SetInput(input)

	postFilter := filters[len(filters)-1].Output()
	log.Debugf("Will pull %v", filters[len(filters)-1])
	filters[len(filters)-1].Pull()

	done := make(chan int)

	go CompareFiltered(t, expected, postFilter, done, true)

	for tok := range LoadTestDocument("chained", strings.Join(inputs, " ")).Tokens() {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
	log.Infof("TestChained Complete")
}

func RunTestCase(testname string, t *testing.T) {
	testcase := TestCases[testname]

	input := NewFilterPipe("test")
	log.Debugf("Created input pipe %v", input)

	filter := testcase.FilterFunc(testname)
	log.Debugf("Created filter %v", filter)
	log.Flush()
	filter.SetInput(input)
	postFilter := filter.Output()

	filter.Pull()

	done := make(chan int)

	go CompareFiltered(t, testcase.Expected, postFilter, done, false)

	testDoc := LoadTestDocument(testname, testcase.Input)
	for tok := range testDoc.Tokens() {
		log.Debugf("Inserting %v into input", tok)
		input.Push(tok)
	}

	close(input.Pipe)
	<-done
	log.Infof("Test[%s] Complete", testname)
}

func TestFilters(t *testing.T) {
	logging.SetupTestLogging()

	for name, _ := range TestCases {
		RunTestCase(name, t)
		/*RunTestCase("hyphens", t)*/
		/*RunTestCase("dates", t)*/
		/*RunTestCase("digits", t)*/
		/*RunTestCase("filenames", t)*/
		/*RunTestCase("email", t)*/
		/*RunTestCase("email", t)*/
	}
}
