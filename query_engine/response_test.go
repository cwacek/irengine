package query_engine

import "sort"
import "testing"

var (
	response1 = &Response{
		[]*Result{
			&Result{"doc1", 2.5, "blah"},
			&Result{"doc2", 2.6, "blah"},
			&Result{"doc3", 2.7, "blah"},
			&Result{"doc4", 2.9, "blah"},
			&Result{"doc5", 2.1, "blah"},
		},
		"",
		"blah",
	}

	ordered = []*Result{
		&Result{"doc4", 2.9, "blah"},
		&Result{"doc3", 2.7, "blah"},
		&Result{"doc2", 2.6, "blah"},
		&Result{"doc1", 2.5, "blah"},
		&Result{"doc5", 2.1, "blah"},
	}

	response2 = &Response{
		[]*Result{
			&Result{"doc1", 3.5, "blah"},
			&Result{"doc7", 3.6, "blah"},
			&Result{"doc8", 3.7, "blah"},
			&Result{"doc2", 3.9, "blah"},
			&Result{"doc3", 3.1, "blah"},
		},
		"",
		"blah",
	}

	combined_ordered = []*Result{
		&Result{"doc8", 3.7, "blah"},
		&Result{"doc7", 3.6, "blah"},
		&Result{"doc4", 2.9, "blah"},
		&Result{"doc3", 2.7, "blah"},
		&Result{"doc2", 2.6, "blah"},
		&Result{"doc1", 2.5, "blah"},
		&Result{"doc5", 2.1, "blah"},
	}
)

func TestResponseExtendUnique(t *testing.T) {

	sort.Sort(response1)

	for i, res := range response1.Results {
		if !res.Equal(ordered[i]) {
			t.Errorf("%v != %v at position %d", res, ordered[i], i)
		}
	}

	response1.ExtendUnique(response2)

	sort.Sort(response1)

	for i, res := range response1.Results {
		if !res.Equal(combined_ordered[i]) {
			t.Errorf("%v != %v at position %d", res, combined_ordered[i], i)
		}
	}

}
