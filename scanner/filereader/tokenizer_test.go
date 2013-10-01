package filereader

import "testing"
import "strings"
import "fmt"
import log "github.com/cihub/seelog"

type testcase struct {
    test string
    expected []Token
}

var (
    tests = []testcase{
        {
            "(7 CFR)",
            []Token{
                {"(", SymbolToken},
                {"7", TextToken},
                {"CFR", TextToken},
                {")", SymbolToken},
            },
        },
        {
            "welcome; this is jim's house. Not jims' house",
            []Token{
                {"welcome", TextToken},
                {"this", TextToken},
                {"is", TextToken},
                {"jims", TextToken},
                {"house", TextToken},
                {"Not", TextToken},
                {"jims", TextToken},
                {"house", TextToken},
            },
        },
        {
            "8:43pm 100.242 100,000,1.10",
            []Token{
                {"8", TextToken},
                {":", SymbolToken},
                {"43pm", TextToken},
                {"100", TextToken},
                {".", SymbolToken},
                {"242", TextToken},
                {"100", TextToken},
                {",", SymbolToken},
                {"000", TextToken},
                {",", SymbolToken},
                {"1", TextToken},
                {".", SymbolToken},
                {"10", TextToken},
            },
        },
        {
            "<PARENT> FR940405-1-00001 </PARENT>",
            []Token{
                {"PARENT", XMLStartToken},
                {"FR940405-1-00001", TextToken},
                {"PARENT", XMLEndToken},
            },
        },
        {
            "<DOCNO>DEADBEEF</DOCNO>",
            []Token{
                {"DOCNO", XMLStartToken},
                {"DEADBEEF", TextToken},
                {"DOCNO", XMLEndToken},
            },
        },
        {
            "<CFRNO>7 CFR Part 28 <!-- blah elsld --></CFRNO>",
            []Token{
                {"CFRNO", XMLStartToken},
                {"7", TextToken},
                {"CFR", TextToken},
                {"Part", TextToken},
                {"28", TextToken},
                {"CFRNO", XMLEndToken},
            },

        },
        {
            "<RINDOCK>[CN&hyph;94&hyph;003] </RINDOCK>",
            []Token{
                {"RINDOCK", XMLStartToken},
                {"[", SymbolToken},
                {"CN-94-003", TextToken},
                {"]", SymbolToken},
                {"RINDOCK", XMLEndToken},
            },

        },
    }
)


func TestTokenizer(t *testing.T) {
    setupLogging("test/seelog.xml")

    for _, test := range tests {
        run_testcase(test, t)
    }
}

func run_testcase(test testcase, t *testing.T) {

    reader := strings.NewReader(test.test)
    tz := BadXMLTokenizer_FromReader(reader)

    i := 0
    for tok := range tz.Tokens() {
        expected := test.expected[i]

        log.Debugf("TEST %d: '%s' == '%s'\n", i, tok, expected)

        if *tok != expected {
            t.Error(fmt.Sprintf("%s != %s at %d\n", tok, expected, i))
        }
        i += 1
        if i > len(test.expected) {
            t.Error(fmt.Sprintf("Tokenizer has more tokens than expected)"))
            break
        }
    }

    if i != len(test.expected) {
        t.Error(fmt.Sprintf("Tokenizer had too few tokens"))
    }
}
