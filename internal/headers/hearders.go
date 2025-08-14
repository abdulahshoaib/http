package headers

import (
	"bytes"
	"fmt"
)

type Headers map[string]string

var crlf = []byte("\r\n")

func NewHeaders() Headers {
	return map[string]string{}
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
		read += idx + len(crlf)

		h[name] = val
	}

	return read, done, nil
}
