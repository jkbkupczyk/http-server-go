package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func prepareTestConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		FileDir: t.TempDir(),
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
			cfg := prepareTestConfig(t)
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

			createFileHandler(cfg, res, req)

			if res.Version != "HTTP/1.1" {
				t.Errorf("invalid http version returned, wanted: 'HTTP/1.1', got: %s", res.Version)
			}
			if res.Status != StatusCreated {
				t.Errorf("invalid http status returned, wanted: '201', got: %d", res.Status)
			}
			if res.Headers == nil {
				t.Errorf("wanted headers to be not nil")
			}

			fPath := filepath.Join(cfg.FileDir, tC.fileName)
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
