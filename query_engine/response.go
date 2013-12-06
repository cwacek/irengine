package query_engine

import log "github.com/cihub/seelog"
import zmq "github.com/pebbe/zmq3"
import "encoding/json"

type Result struct {
	// The document this corresponds to
	Document string
	// The score that document received
	Score float64
	// Random info
	Info string
}

// A sorted set of results
type Response struct {
	Results []*Result
	Error   string
	// The engine that actually answered
	Source string
}

func (r *Result) Equal(o *Result) bool {
	switch {
	case r.Document != o.Document:
		return false
	case r.Score != o.Score:
		return false
	case r.Info != o.Info:
		return false
	default:
		return true
	}
}

func ReceiveResponse(s *zmq.Socket) (r *Response) {
	var msg []byte
	var e error

	if msg, e = s.RecvBytes(0); e != nil {
		panic(e)
	}

	r = new(Response)
	if e = json.Unmarshal(msg, r); e != nil {
		log.Criticalf("Error decoding JSON: %v", e)
		panic(e)
	}

	if r.Results == nil && r.Error == "EMPTYRESULTS" {
		// Decode as empty result set
		r.Results = make([]*Result, 0)
		r.Error = ""
	}

	return r
}

func ErrorResponse(msg string) *Response {
	return &Response{nil, msg, ""}
}

func NewResponse() *Response {
	return &Response{make([]*Result, 0), "", "DEFAULT"}
}

func (r *Response) Send(s *zmq.Socket) {
	var msg []byte
	var e error

	if len(r.Results) == 0 {
		r.Error = "EMPTYRESULTS"
	}

	if msg, e = json.Marshal(r); e != nil {
		panic(e)
	}
	s.SendBytes(msg, 0)
}

func (r *Response) IsError() (string, bool) {
	if r.Results == nil {
		return r.Error, true
	}
	return "", false
}

func (r *Response) Append(result *Result) {
	r.Results = append(r.Results, result)
}

func (r *Response) Extend(result *Response) {
	r.Results = append(r.Results, result.Results...)
}

func (r *Response) ExtendUnique(result *Response) {
	existing := make(map[string]bool)
	for _, res := range r.Results {
		existing[res.Document] = true
	}

	/* Add the ones that aren't already in another dataset */
	for _, res := range result.Results {
		if _, found := existing[res.Document]; !found {
			r.Results = append(r.Results, res)
		}
	}
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
