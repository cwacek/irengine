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

func ReceiveResponse(s *zmq.Socket) (r *Response) {
	var msg []byte
	var e error

	if msg, e = s.RecvBytes(0); e != nil {
		panic(e)
	}

	log.Debugf("Received %s", msg)

	r = new(Response)
	if e = json.Unmarshal(msg, r); e != nil {
		log.Criticalf("Error decoding JSON: %v", e)
		panic(e)
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
