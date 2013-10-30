package query_engine

type Result struct {
	// The document this corresponds to
	Document string
	// The score that document received
	Score float64
}

// A sorted set of results
type Response []*Result

func (r Response) Len() int {
	return len(r)
}

func (r Response) Less(i, j int) bool {
	if r[i].Score > r[j].Score {
		return true
	}

	return false
}

func (r Response) Swap(i, j int) {
	tmp := r[i]
	r[i] = r[j]
	r[j] = tmp
}
