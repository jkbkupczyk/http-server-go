package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
)

const (
	StatusOK                  = 200
	StatusCreated             = 201
	StatusBadRequest          = 400
	StatusNotFound            = 404
	StatusInternalServerError = 500
)

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
	MethodPatch   = "PATCH"
)

const (
	HeaderContentLength   = "Content-Length"
	HeaderContentType     = "Content-Type"
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentEncoding = "Content-Encoding"
)

const EncodingGzip = "gzip"

var (
	ErrCannotReadRequestLine = errors.New("http: cannot read request line")
	ErrInvalidRequestLine    = errors.New("http: invalid request line")
	ErrCannotReadHeaders     = errors.New("http: cannot read headers")
	ErrUnsupportedMethod     = errors.New("http: unsupported method")
	ErrUnsupportedVersion    = errors.New("http: unsupported version")
	ErrInvalidContentLength  = errors.New("http: invalid content length value")
)

type HttpHeaders map[string]string

type HttpRequest struct {
	Method  string
	Target  string
	Version string
	Headers HttpHeaders
	Body    io.Reader
}

type HttpResponse struct {
	Version string
	Status  int
	Headers HttpHeaders
	Body    io.Reader
}

func (req HttpRequest) GetContentLength() int64 {
	cl, ok := req.Headers[HeaderContentLength]
	if !ok {
		return 0
	}

	contentLen, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return 0
	}

	return contentLen
}

func (r *HttpResponse) WriteStr(str string) *HttpResponse {
	r.Body = strings.NewReader(str)
	if r.Headers == nil {
		r.Headers = HttpHeaders{}
	}
	r.Headers[HeaderContentType] = "text/plain"
	return r
}

func Read(r io.Reader) (*HttpRequest, error) {
	br := newBufferedReader(r)

	req := &HttpRequest{}

	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.Join(ErrCannotReadRequestLine, err)
	}
	line = strings.TrimRight(line, "\r\n")

	tokens := strings.Fields(line)
	if len(tokens) != 3 {
		return nil, ErrInvalidRequestLine
	}

	// Status line
	method, version := strings.ToUpper(strings.TrimSpace(tokens[0])), strings.TrimSpace(tokens[2])
	if !methodIsValid(method) {
		return nil, errors.Join(ErrUnsupportedMethod, err)
	}
	if version == "" {
		return nil, errors.Join(ErrUnsupportedVersion, err)
	}

	req.Method = method
	req.Target = strings.TrimSpace(tokens[1])
	req.Version = version

	// Headers
	headers := make(map[string]string)
	for {
		hdrLine, err := br.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
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
	cl, ok := req.Headers[HeaderContentLength]
	if ok {
		value, err := strconv.Atoi(cl)
		if err != nil {
			return nil, errors.Join(ErrInvalidContentLength, err)
		}
		req.Body = io.LimitReader(br, int64(value))
	} else {
		req.Body = io.NopCloser(br)
	}

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

	// in stage 11 (Gzip Compression) CodeCrafters require settings Content-Length
	// header set to the size of the compressed body, which defeats whole purpose of
	// streaming response via Readers/Writers
	var body []byte
	if res.Body != nil {
		if res.Headers[HeaderContentEncoding] == EncodingGzip {
			buff, err := getGzippedBody(res.Body)
			if err != nil {
				return total, err
			}
			body = buff
		} else {
			body, err = io.ReadAll(res.Body)
		}
	}

	// Headers
	if res.Headers != nil {
		res.Headers[HeaderContentLength] = strconv.Itoa(len(body))
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
	nn, err := bw.Write(body)
	total += int64(nn)
	if err != nil {
		return total, err
	}

	return total, bw.Flush()
}

func getGzippedBody(responseBody io.Reader) ([]byte, error) {
	var buff bytes.Buffer
	var err error

	gzipWriter := gzip.NewWriter(&buff)

	if _, err = io.Copy(gzipWriter, responseBody); err != nil {
		return nil, err
	}

	if gzipWriter.Flush() != nil {
		return nil, err
	}

	if gzipWriter.Close() != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

func statusString(code int) string {
	switch code {
	case StatusOK:
		return "OK"
	case StatusCreated:
		return "Created"
	case StatusBadRequest:
		return "Bad Request"
	case StatusNotFound:
		return "Not Found"
	case StatusInternalServerError:
		return "Internal Server Error"
	default:
		return ""
	}
}

func methodIsValid(method string) bool {
	switch method {
	case MethodGet, MethodHead, MethodPost,
		MethodPut, MethodDelete, MethodConnect,
		MethodOptions, MethodTrace, MethodPatch:
		return true
	default:
		return false
	}
}

func newCleanResponse() *HttpResponse {
	return &HttpResponse{
		Version: "HTTP/1.1",
		Status:  200,
		Headers: HttpHeaders{},
	}
}
