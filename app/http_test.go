package main

import (
	"io"
	"strings"
	"testing"
)

func TestReadHeaders(t *testing.T) {
	testCases := []struct {
		desc        string
		source      io.Reader
		wantHeaders HttpHeaders
	}{
		{
			desc:        "read http request - no headers",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\n\r\n"),
			wantHeaders: HttpHeaders{},
		},
		{
			desc:   "read http request - 1 header",
			source: strings.NewReader("GET /index.html HTTP/1.1\r\nUser-Agent: foobar/1.2.3\r\n\r\n"),
			wantHeaders: HttpHeaders{
				"User-Agent": "foobar/1.2.3",
			},
		},
		{
			desc:   "read http request - 2 headers",
			source: strings.NewReader("GET /index.html HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\n\r\n"),
			wantHeaders: HttpHeaders{
				"Host":       "localhost:4221",
				"User-Agent": "curl/7.64.1",
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			req, err := Read(tC.source)
			if err != nil {
				t.Fatalf("wanted no errors but read(io.Reader) returned error: %v", err)
			}

			if req.Headers == nil {
				t.Fatalf("wanted HttpRequest::Headers to not be nil")
			}

			for wantKey, wantValue := range tC.wantHeaders {
				if gotValue := req.Headers[wantKey]; gotValue != wantValue {
					t.Errorf("wanted header[%s] value to be: '%s', but got '%s'", wantKey, wantValue, gotValue)
				}
			}
		})
	}
}

func TestReadRequestLine(t *testing.T) {
	testCases := []struct {
		desc        string
		source      io.Reader
		wantMethod  string
		wantTarget  string
		wantVersion string
	}{
		{
			desc:        "read http request - simple request",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\n\r\n"),
			wantMethod:  "GET",
			wantTarget:  "/index.html",
			wantVersion: "HTTP/1.1",
		},
		{
			desc:        "read http request - complex request",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\nContent-Length: 123\r\n\r\n<html><body><p>Hello</p></body></html>"),
			wantMethod:  "GET",
			wantTarget:  "/index.html",
			wantVersion: "HTTP/1.1",
		},
		{
			desc:        "read http request - full request",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\nHost: localhost:4221\r\nUser-Agent: curl/7.64.1\r\nAccept: */*\r\n\r\n"),
			wantMethod:  "GET",
			wantTarget:  "/index.html",
			wantVersion: "HTTP/1.1",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			req, err := Read(tC.source)
			if err != nil {
				t.Fatalf("wanted no errors but read(io.Reader) returned error: %v", err)
			}
			if tC.wantMethod != req.Method {
				t.Errorf("invalid method, wanted: '%s', got: '%s'", tC.wantMethod, req.Method)
			}
			if tC.wantTarget != req.Target {
				t.Errorf("invalid target, wanted: '%s', got: '%s'", tC.wantTarget, req.Target)
			}
			if tC.wantVersion != req.Version {
				t.Errorf("invalid version, wanted: '%s', got: '%s'", tC.wantVersion, req.Version)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	testCases := []struct {
		desc      string
		res       HttpResponse
		wantValue string
	}{
		{
			desc:      "write empty response",
			res:       HttpResponse{},
			wantValue: " 0 \r\n\r\n",
		},
		{
			desc: "write unsupported status code",
			res: HttpResponse{
				Version: "HTTP/1.1",
				Status:  1234567890,
			},
			wantValue: "HTTP/1.1 1234567890 \r\n\r\n",
		},
		{
			desc: "write full response",
			res: HttpResponse{
				Version: "HTTP/1.1",
				Status:  404,
				Headers: map[string]string{"Content-Type": "text/html; charset=utf-8"},
				Body:    strings.NewReader("Hello, World!"),
			},
			wantValue: "HTTP/1.1 404 Not Found\r\nContent-Type: text/html; charset=utf-8\r\n\r\nHello, World!",
		},
		{
			desc: "write ok response - stage 1",
			res: HttpResponse{
				Version: "HTTP/1.1",
				Status:  200,
			},
			wantValue: "HTTP/1.1 200 OK\r\n\r\n",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			var sb strings.Builder
			_, err := Write(&sb, &tC.res)
			if err != nil {
				t.Fatalf("wanted no errors but write(HttpResponse) returned error: %v", err)
			}
			if str := sb.String(); str != tC.wantValue {
				t.Errorf("invalid value written, wanted: '%s', got: '%s'", tC.wantValue, str)
			}
		})
	}
}

func TestResponseWriteStr(t *testing.T) {
	testCases := []struct {
		desc        string
		str         string
		wantBody    string
		wantBodyLen int
	}{
		{
			desc:        "write empty string",
			str:         "",
			wantBody:    "",
			wantBodyLen: 0,
		},
		{
			desc:        "write blank string",
			str:         " \t   \b ",
			wantBody:    " \t   \b ",
			wantBodyLen: 7,
		},
		{
			desc:        "write non empty string",
			str:         "Hello, World!",
			wantBody:    "Hello, World!",
			wantBodyLen: 13,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			res := &HttpResponse{
				Version: "HTTP/1.1",
				Status:  200,
			}

			res = res.WriteStr(tC.str)

			if res.Headers["Content-Type"] != "text/plain" {
				t.Errorf("missing or invalid 'Content-Type' header value, wanted: 'text/plain', got: '%s'", res.Headers["Content-Type"])
			}
			if l := res.BodyLength(); l != tC.wantBodyLen {
				t.Errorf("invalid body length ('Content-Length'), wanted: '%d', got: '%d'", tC.wantBodyLen, l)
			}
			sb := new(strings.Builder)
			io.Copy(sb, res.Body)
			if s := sb.String(); s != tC.wantBody {
				t.Errorf("invalid body contents, wanted: '%s', got: '%s'", tC.wantBody, s)
			}
		})
	}
}

func TestNewCleanResponse(t *testing.T) {
	cr := newCleanResponse()

	if cr.Version != "HTTP/1.1" {
		t.Errorf("wanted Version to be 'HTTP/1.1' but got: %s", cr.Version)
	}
	if cr.Status != StatusOK {
		t.Errorf("wanted Status to be OK but got: %d", cr.Status)
	}
	if cr.Headers == nil {
		t.Errorf("wanted Headers to not be nil but got: %s", cr.Headers)
	}
	if cr.Body != nil {
		t.Errorf("wanted Body to be nil but got non-nil value")
	}
	if cr.BodyLength != nil {
		t.Errorf("wanted BodyLength to be nil but got non-nil value")
	}
}
