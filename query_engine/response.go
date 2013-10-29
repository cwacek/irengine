package query_engine

type Result struct {
	// The document this corresponds to
	Document string
	// The score that document received
	Score float64
}

// A sorted set of results
type Response []Result
