package main

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
)

var (
	ErrCannotReadRequestLine = errors.New("http: cannot read request line")
	ErrInvalidRequestLine    = errors.New("http: invalid request line")
	ErrCannotReadHeaders     = errors.New("http: cannot read headers")
)

type HttpHeaders map[string]string

type HttpRequest struct {
	Method  string
	Target  string
	Version string
	Headers HttpHeaders
	Body    io.Reader
}

type BodyLengthFunc func() int

type HttpResponse struct {
	Version    string
	Status     int
	Headers    HttpHeaders
	Body       io.Reader
	BodyLength BodyLengthFunc
}

func Read(r io.Reader) (*HttpRequest, error) {
	br := newBufferedReader(r)

	req := &HttpRequest{}

	line, err := br.ReadString('\n')
	if err != nil {
		return nil, errors.Join(ErrCannotReadRequestLine, err)
	}

	tokens := strings.SplitN(line, " ", 3)
	if len(tokens) < 3 {
		return nil, ErrInvalidRequestLine
	}

	// Status line
	req.Method = strings.TrimSpace(tokens[0])
	req.Target = strings.TrimSpace(tokens[1])
	req.Version = strings.TrimSpace(tokens[2])

	// Headers
	headers := make(map[string]string)
	for {
		hdrLine, err := br.ReadString('\n')
		if err != nil {
			return nil, errors.Join(ErrCannotReadHeaders, err)
		}

		hdrLine = strings.TrimSpace(hdrLine)
		if hdrLine == "" {
			break
		}

		headerValues := strings.SplitN(hdrLine, ":", 2)
		key := strings.TrimSpace(headerValues[0])
		value := strings.TrimSpace(headerValues[1])

		headers[key] = value
	}
	req.Headers = headers

	// Body
	req.Body = r

	return req, nil
}

func Write(w io.Writer, res *HttpResponse) (int64, error) {
	bw := newBufferedWriter(w)
	total := int64(0)

	// Status line
	n, err := fmt.Fprintf(bw, "%s %d %s\r\n", res.Version, res.Status, statusString(res.Status))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Headers
	if res.Headers != nil {
		if res.BodyLength != nil {
			res.Headers["Content-Length"] = strconv.Itoa(res.BodyLength())
		}
		// Write headers in alphabetical order
		for _, k := range slices.Sorted(maps.Keys(res.Headers)) {
			nn, err := fmt.Fprintf(bw, "%s: %s\r\n", k, res.Headers[k])
			total += int64(nn)
			if err != nil {
				return total, err
			}
		}
	}

	n, err = bw.WriteString("\r\n")
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Body
	if res.Body != nil {
		nn, err := io.Copy(bw, res.Body)
		total += nn
		if err != nil {
			return total, err
		}
	}

	return total, bw.Flush()
}

func statusString(code int) string {
	switch code {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}

func newCleanResponse() *HttpResponse {
	return &HttpResponse{
		Version: "HTTP/1.1",
		Status:  200,
		Headers: HttpHeaders{},
	}
}
