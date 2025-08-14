package request

import (
	"bytes"
	"fmt"
	"http/internal/headers"
	"io"
	"strconv"
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

// ENUM
const (
	Init    ParserState = "init"
	Done    ParserState = "done"
	Error   ParserState = "error"
	Headers ParserState = "headers"
	Body    ParserState = "body"
)

type Request struct {
	RequestLine RequestLine
	State       ParserState
	Headers     *headers.Headers
	Body        string
}

func getInt(headers *headers.Headers, name string, defaultVal int) int {
	value, exists := headers.Get(name)
	if !exists {
		return defaultVal
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}
	return val
}

func (r *Request) hasBody() bool {
	contentlen := getInt(r.Headers, "content-length", 0)
	return contentlen > 0
}

func (r *Request) parse(data []byte) (int, error) {
	read := 0
outer:
	for {
		currData := data[read:]
		if len(currData) == 0 {
			break outer
		}

		switch r.State {
		case Error:
			return 0, ERROR_STATE
		case Init:
			rl, n, err := parseRequestLine(currData)
			if err != nil {
				r.State = Error
				return 0, err
			}

			if n == 0 {
				break outer
			}

			r.RequestLine = *rl
			read += n

			r.State = Headers

		case Headers:
			n, done, err := r.Headers.Parse(currData)
			if err != nil {
				r.State = Error
				return 0, err
			}

			if n == 0 {
				break outer
			}

			read += n

			if done {
				if r.hasBody() {
					r.State = Body
				} else {
					r.State = Done
				}
			}

		case Body:
			contentlen := getInt(r.Headers, "content-length", 0)
			if contentlen == 0 {
				panic("chuncked encoding not implemented")
			}

			remaining := min(contentlen-len(r.Body), len(currData))
			r.Body += string(currData[:remaining])
			read += remaining

			if len(r.Body) == contentlen {
				r.State = Done
			}

		case Done:
			break outer

		default:
			panic("Unidentifiable State")
		}
	}
	return read, nil
}

func (r *Request) parseDone() bool {
	return r.State == Done
}

func (r *Request) parseError() bool {
	return r.State == Error
}

func newRequest() *Request {
	return &Request{
		State:   Init,
		Headers: headers.NewHeaders(),
		Body:    "",
	}
}

var MALFORMED_REQ_LINE = fmt.Errorf("malformed request-line")
var UNSUPPORTED_HTTP_VER = fmt.Errorf("unsupported http version (strictly has to be 1.1)")
var INCORRECT_METHOD = fmt.Errorf("incorrect request-line method")
var ERROR_STATE = fmt.Errorf("request in error State")

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
