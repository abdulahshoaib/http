package request

import (
	"bytes"
	"fmt"
	"io"
)

type RequestLine struct {
	Method        string
	RequestTarget string
	HttpVersion   string
}

func (r *RequestLine) ValidHTTPVersion() bool {
	return r.HttpVersion == "HTTP/1.1"
}

func (r *RequestLine) ValidMethod() bool {
	switch r.Method {
	case "GET", "PUT", "POST", "PATCH", "DELETE", "OPTIONS":
		return true
	default:
		return false
	}
}

type ParserState string

const (
	Init ParserState = "init"
	Done ParserState = "done"
	Error ParserState = "error"
)

type Request struct {
	RequestLine RequestLine
	state       ParserState
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		switch r.state {
		case Error:
			return 0, ERROR_STATE
		case Init:
			rl, n, err := parseRequestLine(data[read:])
			if err != nil {
				r.state = Error
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n

			r.state = Done

		case Done:
			break outer
		}
	}
	return read, nil
}

func (r *Request) parseDone() bool {
	return r.state == Done
}

func (r *Request) parseError() bool {
	return r.state == Error
}

func newRequest() *Request {
	return &Request{
		state: Init,
	}
}

var MALFORMED_REQ_LINE = fmt.Errorf("malformed request-line")
var UNSUPPORTED_HTTP_VER = fmt.Errorf("unsupported http version (strictly has to be 1.1)")
var INCORRECT_METHOD = fmt.Errorf("incorrect request-line method")
var ERROR_STATE = fmt.Errorf("request in error state")

var SEPERATOR = []byte("\r\n")

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, SEPERATOR)
	if idx == -1 {
		return nil, 0, nil
	}

	startLine := b[:idx]
	read := idx + len(SEPERATOR)

	parts := bytes.Split(startLine, []byte(" "))
	if len(parts) != 3 {
		return nil, 0, MALFORMED_REQ_LINE
	}

	httpPart := bytes.Split(parts[2], []byte("/"))
	if len(httpPart) != 2 || string(httpPart[0]) != "HTTP" || string(httpPart[1]) != "1.1" {
		return nil, 0, MALFORMED_REQ_LINE
	}

	rl := &RequestLine{
		Method:        string(parts[0]),
		RequestTarget: string(parts[1]),
		HttpVersion:   string(httpPart[1]),
	}

	return rl, read, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := newRequest()

	buf := make([]byte, 1024)
	idx := 0
	for !request.parseDone() && !request.parseError() {
		n, err := reader.Read(buf[idx:])
		if err != nil {
			return nil, err
		}

		idx += n
		readn, err := request.parse(buf[:idx])
		if err != nil {
			return nil, err
		}

		copy(buf, buf[readn:idx])
		idx -= readn
	}

	return request, nil
}
