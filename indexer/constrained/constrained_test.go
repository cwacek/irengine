package constrained

import "testing"
import "io/ioutil"
import "encoding/gob"
import "os"
import "bytes"
import "io"
import "strings"
import index "github.com/cwacek/irengine/indexer"
import "github.com/cwacek/irengine/indexer/filters"
import "github.com/cwacek/irengine/scanner/filereader"
import log "github.com/cihub/seelog"
import "github.com/cwacek/irengine/logging"

var (
	serialized_pls = `
    james bond # 3 2
    that # 3 2 3 4
    that # 1 17
    that # 5 12
    there # 3 1 5
    which # 1 15
    which # 5 12 15
    `
	serialized_basic_pls = `that # 1 1
that # 3 3
that # 5 1
there # 3 2
which # 1 1
which # 5 2
`

	expected_pl = map[string]string{
		"james bond":  "3 2" ,
		"that":  "1 17 | 3 2 3 4 | 5 12" ,
		"there": "3 1 5" ,
		"which": "1 15 | 5 12 15" ,
	}

	expected_basic_pl = map[string]string{
		"that":  "1 1 | 3 3 | 5 1" ,
		"there": "3 2" ,
		"which": "1 1 | 5 2" ,
	}

    reserialized_pls = []byte(
        "james bond # 3 2\n" +
        "that # 1 17\n" +
        "that # 3 2 3 4\n" +
        "that # 5 12\n" +
        "there # 3 1 5\n" +
        "which # 1 15\n" +
        "which # 5 12 15\n")

        testDocs = []filereader.Document{
            filters.LoadTestDocument("A01",
            "The quick brown fox"),
            filters.LoadTestDocument("A02",
            "The slight brown dog"),
            filters.LoadTestDocument("A03",
            "Here dog. Here doggie dog dog"),
        }

  term string
  basic_pl index.PostingList
)

func init() {
  basic_pl = index.PositionalPostingListInitializer.Create()

  for i := 0; i < 500; i++ {
    switch {
    case i % 4 == 0:
      term = "4"
    case i % 7 == 0:
      term = "7"
    default:
      term = "2"
    }
    basic_pl.InsertRawEntry(term, filereader.DocumentId(i), i)
  }
}

func TestPostingListSetSerialization(t *testing.T) {
    logging.SetupTestLogging()

    log.Info("Starting PLS Serialization test")
	pls := NewPostingListSet("testStore", index.PositionalPostingListInitializer)

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

func TestPostingListSetBasicSerialize(t *testing.T) {
    logging.SetupTestLogging()

    log.Info("Starting PLS Serialization test")
	pls := NewPostingListSet("testStore", index.BasicPostingListInitializer)

	pls.Load(strings.NewReader(serialized_basic_pls))

	for term, exp := range expected_basic_pl {

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

    for i, v := range buf.Bytes() {
      switch {
      case i >= len(serialized_basic_pls):
        t.Errorf("Found end of expected value, but dump had %c at %d", v, i)

      case v != serialized_basic_pls[i]:
        t.Errorf("'%c' did not match expected '%c' at %d", v, serialized_basic_pls[i], i)
    }
  }
    log.Info("Completed")
}

func TestLRU(t *testing.T) {
    logging.SetupTestLogging()

    lex := new(lexicon)

    lex.lru_cache = LRUSet{
        NewPLSContainer(NewPostingListSet("1", index.PositionalPostingListInitializer)),
        NewPLSContainer(NewPostingListSet("2", index.PositionalPostingListInitializer)),
        NewPLSContainer(NewPostingListSet("3", index.PositionalPostingListInitializer)),
        NewPLSContainer(NewPostingListSet("4", index.PositionalPostingListInitializer)),
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

func BenchmarkSerialize(b *testing.B) {
  tmp, _ := ioutil.TempFile("/tmp","ser")
  tmpInfo, _ := tmp.Stat()
  defer os.Remove(tmpInfo.Name())
  var it index.PostingListIterator

  for i := 0 ;i < b.N; i++ {
    for it = basic_pl.Iterator(); it.Next(); {
      io.WriteString(tmp, it.Value().Serialize())
    }
  }
}

func BenchmarkSerializeTo(b *testing.B) {
  tmp, _ := ioutil.TempFile("/tmp","serto")
  tmpInfo, _ := tmp.Stat()
  defer os.Remove(tmpInfo.Name())
  var it index.PostingListIterator

  for i := 0 ;i < b.N; i++ {
    for it = basic_pl.Iterator(); it.Next(); {
      it.Value().SerializeTo(tmp)
    }
  }
}

func BenchmarkGobbing(b *testing.B) {
  tmp, _ := ioutil.TempFile("/tmp","gob")
  tmpInfo, _ := tmp.Stat()
  defer os.Remove(tmpInfo.Name())
  gobber := gob.NewEncoder(tmp)
  var it index.PostingListIterator

  for i := 0 ;i < b.N; i++ {
    for it = basic_pl.Iterator(); it.Next(); {
      gobber.Encode(it.Value())
    }
  }

}

func BenchmarkLoadDump(b *testing.B) {
    logging.SetupTestLogging()
    var pls *PostingListSet
    tmp, _ := ioutil.TempFile("/tmp","blah")
    tmpInfo, _ := tmp.Stat()
    defer os.Remove(tmpInfo.Name())
    /*log.Infof("Benching using %s as temp file", tmpInfo.Name())*/

    pls = NewPostingListSet("testStore", index.PositionalPostingListInitializer)
    file, _ := os.Open("testpls.txt")
    pls.Load(file)

    for i:= 0; i < b.N; i++ {
      pls.Dump(tmp)
      pls = nil
      pls = NewPostingListSet("testStore", index.PositionalPostingListInitializer)
      pls.Load(tmp)
    }

}
