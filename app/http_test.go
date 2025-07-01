package main

import (
	"io"
	"strings"
	"testing"
)

func TestReadRequestLine(t *testing.T) {
	testCases := []struct {
		desc        string
		source      io.Reader
		wantMethod  string
		wantTarget  string
		wantVersion string
	}{
		{
			desc:        "read simple request",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\n\r\n"),
			wantMethod:  "GET",
			wantTarget:  "/index.html",
			wantVersion: "HTTP/1.1",
		},
		{
			desc:        "read complex request",
			source:      strings.NewReader("GET /index.html HTTP/1.1\r\nContent-Length: 123\r\n\r\n<html><body><p>Hello</p></body></html>"),
			wantMethod:  "GET",
			wantTarget:  "/index.html",
			wantVersion: "HTTP/1.1",
		},
		{
			desc:        "read request ",
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
			_, err := Write(&sb, tC.res)
			if err != nil {
				t.Fatalf("wanted no errors but write(HttpResponse) returned error: %v", err)
			}
			if str := sb.String(); str != tC.wantValue {
				t.Errorf("invalid value written, wanted: '%s', got: '%s'", tC.wantValue, str)
			}
		})
	}
}
