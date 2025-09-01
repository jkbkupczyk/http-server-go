package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func newMockApp(t *testing.T) *app {
	t.Helper()
	return &app{
		cfg: Config{
			FileDir: t.TempDir(),
		},
		log: slog.New(NewNoopHandler()),
	}
}

func readerToString(t *testing.T, r io.Reader) string {
	t.Helper()
	buf := new(strings.Builder)
	io.Copy(buf, r)
	return buf.String()
}

func TestRootHandler(t *testing.T) {
	app := newMockApp(t)
	res := &HttpResponse{}
	req := &HttpRequest{}

	app.rootHandler(res, req)

	if res.Status != StatusOK {
		t.Errorf("expected status code to be '200', got: '%d'", res.Status)
	}
}

func TestEchoHandler(t *testing.T) {
	testCases := []struct {
		desc       string
		req        *HttpRequest
		wantStatus int
		wantBody   string
	}{
		{
			desc: "empty target",
			req: &HttpRequest{
				Target: "",
			},
			wantStatus: StatusOK,
			wantBody:   "",
		},
		{
			desc: "echo - empty",
			req: &HttpRequest{
				Target: "/echo/",
			},
			wantStatus: StatusOK,
			wantBody:   "",
		},
		{
			desc: "echo - blank",
			req: &HttpRequest{
				Target: "/echo/ ",
			},
			wantStatus: StatusOK,
			wantBody:   " ",
		},
		{
			desc: "echo - ok",
			req: &HttpRequest{
				Target: "/echo/hello world!",
			},
			wantStatus: StatusOK,
			wantBody:   "hello world!",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			app := newMockApp(t)
			res := &HttpResponse{}

			app.echoHandler(res, tC.req)

			if res.Status != StatusOK {
				t.Errorf("invalid http status returned, wanted: '200', got: %d", res.Status)
			}
			if gotBody := readerToString(t, res.Body); gotBody != tC.wantBody {
				t.Errorf("invalid body returned, wanted: '%s', got: '%s'", tC.wantBody, gotBody)
			}
		})
	}
}

func TestCreateFileHandler(t *testing.T) {
	testCases := []struct {
		desc         string
		fileName     string
		fileContents string
	}{
		{
			desc:         "should create empty file",
			fileName:     "empty",
			fileContents: "",
		},
		{
			desc:         "should create blank file",
			fileName:     "blank",
			fileContents: " \t",
		},
		{
			desc:         "should create file with contents",
			fileName:     "hello",
			fileContents: "Hello, World!",
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			app := newMockApp(t)
			res := newCleanResponse()
			req := &HttpRequest{
				Method:  MethodPost,
				Target:  "/files/" + tC.fileName,
				Version: "HTTP/1.1",
				Headers: HttpHeaders{
					"Host":           "localhost:4221",
					"User-Agent":     "curl/7.64.1",
					"Accept":         "*/*",
					"Content-Type":   "application/octet-stream",
					"Content-Length": strconv.Itoa(len(tC.fileContents)),
				},
				Body: strings.NewReader(tC.fileContents),
			}

			app.createFileHandler(res, req)

			if res.Version != "HTTP/1.1" {
				t.Errorf("invalid http version returned, wanted: 'HTTP/1.1', got: %s", res.Version)
			}
			if res.Status != StatusCreated {
				t.Errorf("invalid http status returned, wanted: '201', got: %d", res.Status)
			}
			if res.Headers == nil {
				t.Errorf("wanted headers to be not nil")
			}

			fPath := filepath.Join(app.cfg.FileDir, tC.fileName)
			buff, err := os.ReadFile(fPath)
			if err != nil {
				t.Fatalf("could not read file: %v", err)
			}
			if fileContents := string(buff); fileContents != tC.fileContents {
				t.Errorf("file '%s' contents differ, wanted: '%s', got: %s", fPath, tC.fileContents, fileContents)
			}
		})
	}
}

type noopLogger struct {
}

func NewNoopHandler() slog.Handler {
	return &noopLogger{}
}

func (noopLogger) Enabled(context.Context, slog.Level) bool {
	return false
}

func (noopLogger) Handle(context.Context, slog.Record) error {
	return nil
}

func (h noopLogger) WithAttrs([]slog.Attr) slog.Handler {
	return &noopLogger{}
}

func (h noopLogger) WithGroup(string) slog.Handler {
	return &noopLogger{}
}
