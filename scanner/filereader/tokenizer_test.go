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
                Token{Text: "7", Type:  TextToken},
                Token{Text: "CFR", Type:  TextToken},
            },
        },
        {
            "welcome; this is jim's house. Not jims' house",
            []Token{
                Token{Text: "welcome", Type:  TextToken},
                Token{Text: "this", Type:  TextToken},
                Token{Text: "is", Type:  TextToken},
                Token{Text: "jims", Type:  TextToken},
                Token{Text: "house", Type:  TextToken},
                Token{Text: "Not", Type:  TextToken},
                Token{Text: "jims", Type:  TextToken},
                Token{Text: "house", Type:  TextToken},
            },
        },
        {
            "8:43pm 100.242 100,000,1.10",
            []Token{
                Token{Text: "8:43pm", Type:  TextToken},
                Token{Text: "100.242", Type:  TextToken},
                Token{Text: "100,000,1.10", Type:  TextToken},
            },
        },
        {
            "<PARENT> FR940405-1-00001 </PARENT>",
            []Token{
                Token{Text: "PARENT", Type:  XMLStartToken},
                Token{Text: "FR940405-1-00001", Type:  TextToken},
                Token{Text: "PARENT", Type:  XMLEndToken},
            },
        },
        {
            "<DOCNO>DEADBEEF</DOCNO>",
            []Token{
                Token{Text: "DOCNO", Type:  XMLStartToken},
                Token{Text: "DEADBEEF", Type:  TextToken},
                Token{Text: "DOCNO", Type:  XMLEndToken},
            },
        },
        {
            "<CFRNO>7 $CFR Part£ 28 <!-- blah elsld --></CFRNO>",
            []Token{
                Token{Text: "CFRNO", Type:  XMLStartToken},
                Token{Text: "7", Type:  TextToken},
                Token{Text: "$CFR", Type:  TextToken},
                Token{Text: "Part£", Type:  TextToken},
                Token{Text: "28", Type:  TextToken},
                Token{Text: "CFRNO", Type:  XMLEndToken},
            },

        },
        {
            "<RINDOCK>[CN&hyph;94&hyph;003] </RINDOCK>",
            []Token{
                Token{Text: "RINDOCK", Type:  XMLStartToken},
                Token{Text: "CN-94-003", Type:  TextToken},
                Token{Text: "RINDOCK", Type:  XMLEndToken},
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
            t.Error(fmt.Sprintf("Tokenizer has more tokens (%d) than expected (%d)",
            i, len(test.expected)))
            break
        }
    }

    if i != len(test.expected) {
        t.Error(fmt.Sprintf("Tokenizer had too few tokens"))
    }
}
