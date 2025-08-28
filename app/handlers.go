package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func rootHandler(res *HttpResponse, _ *HttpRequest) {
	res.Status = StatusOK
}

func echoHandler(res *HttpResponse, req *HttpRequest) {
	res.Status = StatusOK
	echo, _ := strings.CutPrefix(req.Target, "/echo/")
	res.WriteStr(echo)
}

func userAgentHandler(res *HttpResponse, req *HttpRequest) {
	res.Status = StatusOK
	res.WriteStr(req.Headers["User-Agent"])
}

func filesHandler(cfg Config, res *HttpResponse, req *HttpRequest) {
	switch req.Method {
	case MethodGet:
		readFileHandler(cfg, res, req)
	case MethodPost:
		createFileHandler(cfg, res, req)
	}
}

func readFileHandler(cfg Config, res *HttpResponse, req *HttpRequest) {
	fileName, ok := strings.CutPrefix(req.Target, "/files/")
	if fileName == "" || !ok {
		res.Status = StatusBadRequest
		return
	}

	f, err := os.Open(filepath.Join(cfg.FileDir, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			res.Status = StatusNotFound
			return
		}
		res.Status = StatusInternalServerError
		resp := fmt.Sprintf("Could not load file: %s", err.Error())
		res.WriteStr(resp)
		return
	}

	res.Status = StatusOK
	res.Body = f
	res.Headers[HeaderContentType] = "application/octet-stream"
}

func createFileHandler(cfg Config, res *HttpResponse, req *HttpRequest) {
	fileName, _ := strings.CutPrefix(req.Target, "/files/")

	if err := os.MkdirAll(cfg.FileDir, os.ModePerm); err != nil {
		fmt.Printf("Could not create dirs: %s\n", err.Error())
		res.Status = StatusInternalServerError
		res.WriteStr("Could not create dirs: " + err.Error())
		return
	}

	filePath := filepath.Join(cfg.FileDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Could not create file: %s\n", err.Error())
		res.Status = StatusInternalServerError
		res.WriteStr("Could not create file: " + err.Error())
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	n, err := io.Copy(w, req.Body)
	if err != nil {
		fmt.Printf("Could not write data to file: %s\n", err.Error())
		res.Status = StatusInternalServerError
		res.WriteStr("Could not write data to file: " + err.Error())
		return
	}
	w.Flush()

	fmt.Printf("Written %d bytes to file %s\n", n, fileName)

	res.Status = StatusCreated
}
