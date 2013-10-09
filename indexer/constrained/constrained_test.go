package constrained

import "testing"
import "io/ioutil"
/*import "os"*/
import "bytes"
import "strings"
import index "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/logging"

var (
	serialized_pls = `
    that A03 2,3,4
    that A01 17
    that A05 12
    there A03 1,5
    which A01 15
    which A05 12,15
    `

	expected_pl = map[string]string{
		"that":  "A01 17 | A03 2,3,4 | A05 12",
		"there": "A03 1,5",
		"which": "A01 15 | A05 12,15",
	}

    reserialized_pls = []byte(
        "that A01 17\n" +
        "that A03 2,3,4\n" +
        "that A05 12\n" +
        "there A03 1,5\n" +
        "which A01 15\n" +
        "which A05 12,15\n")

        testDocs = []filereader.Document{
            filters.LoadTestDocument("A01",
            "The quick brown fox"),
            filters.LoadTestDocument("A02",
            "The slight brown dog"),
            filters.LoadTestDocument("A03",
            "Here dog. Here doggie dog dog"),
        }

)

func TestPostingListSetSerialization(t *testing.T) {
    logging.SetupTestLogging()

    log.Info("Starting PLS Serialization test")
	pls := NewPostingListSet("testStore", index.NewPositionalPostingList)

	pls.Load(strings.NewReader(serialized_pls))

	for term, exp := range expected_pl {

        pl, ok := pls.listMap[term]
        switch {

            case ! ok:
            t.Errorf("PL didn't contain term '%s'", term)

            case pl.String() != exp:
			t.Errorf("PL for '%s': '%s'. did not match expected '%s'", term, pl.String(), exp)
        }
	}

    buf := new(bytes.Buffer)
    pls.Dump(buf)

    if !bytes.Equal(buf.Bytes(), reserialized_pls) {
        t.Errorf(`Reserialized bytes differ. 
Wanted:
%s
Got:
%s`, reserialized_pls, buf.String())
    }
    log.Info("Completed")
}

func TestLRU(t *testing.T) {
    logging.SetupTestLogging()

    lex := new(lexicon)

    lex.lru_cache = LRUSet{
        NewPostingListSet("1", index.NewPositionalPostingList),
        NewPostingListSet("2", index.NewPositionalPostingList),
        NewPostingListSet("3", index.NewPositionalPostingList),
        NewPostingListSet("4", index.NewPositionalPostingList),
    }

    if lrutag := lex.lru_cache.LeastRecent().Tag; lrutag != "1" {
        t.Errorf("LRU was '%s', but expected '1'",lrutag)
    }

    lex.makeRecent(lex.lru_cache[0])

    if lrutag := lex.lru_cache.LeastRecent().Tag; lrutag != "2" {
        t.Errorf("LRU was '%s', but expected '2'",lrutag)
    }

    exp_tags := []DatastoreTag{"2","3","4","1"}

    for i, pls := range lex.lru_cache {
        if exp_tags[i] != pls.Tag {
            t.Errorf("Expected '%s' but found '%s'",exp_tags[i], pls.Tag)
        }
    }


    t.Log("Test ReplaceOldest functionality")
    exp_tags = []DatastoreTag{"3","4","1"}

    t.Log("LRU Before: %v", lex.lru_cache)
    lex.lru_cache = lex.lru_cache.RemoveOldest()
    t.Log("LRU After: %v", lex.lru_cache)

    for i, pls := range lex.lru_cache {
        if exp_tags[i] != pls.Tag {
            t.Errorf("Expected '%s' but found '%s'",exp_tags[i], pls.Tag)
        }
    }
}

func TestConstrainedMemory(t *testing.T) {
    logging.SetupTestLogging()
    var tmpDir string
    var err error

    if tmpDir, err = ioutil.TempDir("", "irtest"); err != nil {
        t.Errorf("Error creating temp dir %v", err)
        return
    }
    log.Infof("Using temporary directory: %s", tmpDir)

    lex := NewLexicon(12, tmpDir)

    for _, document := range testDocs {
        log.Debugf("Inserting %s", document)
        for token := range document.Tokens() {
            lex.InsertToken(token)
        }
        log.Debugf("Finished inserting %s", document)
    }

    lex.(index.PersistentLexicon).SaveToDisk()

    /*os.RemoveAll(tmpDir)*/
}
