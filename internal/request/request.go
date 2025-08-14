package request

import (
	"errors"
	"fmt"
	"io"
	"strings"
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

type Request struct {
	RequestLine RequestLine
}

var MALFORMED_REQ_LINE = fmt.Errorf("malformed request-line")
var UNSUPPORTED_HTTP_VER = fmt.Errorf("unsupported http version (strictly has to be 1.1)")
var INCORRECT_METHOD = fmt.Errorf("incorrect request-line method")
var INCOMPLETE_START_LINE = fmt.Errorf("incomplete start line")
var SEPERATOR = "\r\n"

func parseRequestLine(str string) (*RequestLine, string, error) {
	idx := strings.Index(str, SEPERATOR)
	if idx == -1 {
		return nil, str, INCOMPLETE_START_LINE
	}

	startLine := str[:idx]
	restOfMsg := str[idx+len(SEPERATOR):]

	parts := strings.Split(startLine, " ")
	if len(parts) != 3 {
		return nil, restOfMsg, MALFORMED_REQ_LINE
	}

	httpPart := strings.Split(parts[2], "/")
	if len(httpPart) != 2 || httpPart[0] != "HTTP" || httpPart[1] != "1.1" {
		return nil, restOfMsg, MALFORMED_REQ_LINE
	}

	rl := &RequestLine{
		Method:        parts[0],
		RequestTarget: parts[1],
		HttpVersion:   httpPart[1],
	}

	return rl, restOfMsg, nil
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to read io.ReadlAll"), err)
	}

	str := string(data)
	rl, _, err := parseRequestLine(str)
	if err != nil {
		return nil, err
	}

	return &Request{
		RequestLine: *rl,
	}, err
}
