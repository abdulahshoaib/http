package headers

import (
	"bytes"
	"fmt"
	"strings"
)

func isValidToken(str string) bool {
	for _, ch := range str {
		res := false
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			res = true
		}
		switch ch {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			res = true
		}

		if !res {
			return false
		}
	}

	return true
}

var crlf = []byte("\r\n")

func NewHeaders() *Headers {
	return &Headers{
		headers: map[string]string{},
	}
}

func parseHeader(fieldLines []byte) (string, string, error) {
	parts := bytes.SplitN(fieldLines, []byte(":"), 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed field-line")
	}

	name := parts[0]
	val := bytes.TrimSpace(parts[1])

	if bytes.HasSuffix(name, []byte(" ")) {
		return "", "", fmt.Errorf("malformed field-name")
	}

	return string(name), string(val), nil
}

type Headers struct {
	headers map[string]string
}

func (h Headers) Get(name string) (string, bool) {
	str, exists := h.headers[strings.ToLower(name)]
	return str, exists
}

func (h Headers) Set(name, value string) {
	name = strings.ToLower(name)

	if prev, exists := h.headers[name]; exists {
		h.headers[name] = fmt.Sprintf("%s,%s", prev, value)
	} else {
		h.headers[name] = value
	}
}

func (h Headers) ForEach(cb func(n, v string)) {
	for n, v := range h.headers {
		cb(n, v)
	}
}

func (h Headers) Parse(data []byte) (int, bool, error) {

	read := 0
	done := false
	for {
		idx := bytes.Index(data[read:], crlf)
		if idx == -1 {
			break
		}

		// empty header
		if idx == 0 {
			done = true
			read += len(crlf)
			break
		}

		name, val, err := parseHeader(data[read : read+idx])
		if err != nil {
			return 0, false, err
		}
		if !isValidToken(name) {
			return 0, false, fmt.Errorf("malformed header name")
		}

		read += idx + len(crlf)

		h.Set(name, val)
	}

	return read, done, nil
}
