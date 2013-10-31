package query_engine

type Result struct {
	// The document this corresponds to
	Document string
	// The score that document received
	Score float64
}

// A sorted set of results
type Response struct {
	Results []*Result
	Error   string
}

func ErrorResponse(msg string) *Response {
	return &Response{nil, msg}
}

func NewResponse() *Response {
	return &Response{make([]*Result, 0), ""}
}

func (r *Response) Append(result *Result) {
	r.Results = append(r.Results, result)
}

func (r Response) Len() int {
	return len(r.Results)
}

func (r Response) Less(i, j int) bool {
	if r.Results[i].Score > r.Results[j].Score {
		return true
	}

	return false
}

func (r Response) Swap(i, j int) {
	tmp := r.Results[i]
	r.Results[i] = r.Results[j]
	r.Results[j] = tmp
}
