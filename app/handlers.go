package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func (a *app) notFoundHandler(res *HttpResponse, _ *HttpRequest) {
	res.Status = StatusNotFound
}

func (a *app) rootHandler(res *HttpResponse, _ *HttpRequest) {
	res.Status = StatusOK
}

func (a *app) echoHandler(res *HttpResponse, req *HttpRequest) {
	res.Status = StatusOK
	echo, _ := strings.CutPrefix(req.Target, "/echo/")
	res.WriteStr(echo)
}

func (a *app) userAgentHandler(res *HttpResponse, req *HttpRequest) {
	res.Status = StatusOK
	res.WriteStr(req.Headers["User-Agent"])
}

func (a *app) filesHandler(res *HttpResponse, req *HttpRequest) {
	switch req.Method {
	case MethodGet:
		a.readFileHandler(res, req)
	case MethodPost:
		a.createFileHandler(res, req)
	}
}

func (a *app) readFileHandler(res *HttpResponse, req *HttpRequest) {
	fileName, ok := strings.CutPrefix(req.Target, "/files/")
	if fileName == "" || !ok {
		res.Status = StatusBadRequest
		return
	}

	f, err := os.Open(filepath.Join(a.cfg.FileDir, fileName))
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

func (a *app) createFileHandler(res *HttpResponse, req *HttpRequest) {
	fileName, _ := strings.CutPrefix(req.Target, "/files/")

	if err := os.MkdirAll(a.cfg.FileDir, os.ModePerm); err != nil {
		a.log.Warn("could not create dirs", slog.String("error", err.Error()))
		res.Status = StatusInternalServerError
		res.WriteStr("Could not create dirs: " + err.Error())
		return
	}

	filePath := filepath.Join(a.cfg.FileDir, fileName)
	f, err := os.Create(filePath)
	if err != nil {
		a.log.Warn("could not create file", slog.String("fileName", fileName), slog.String("error", err.Error()))
		res.Status = StatusInternalServerError
		res.WriteStr("Could not create file: " + err.Error())
		return
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	n, err := io.Copy(w, req.Body)
	if err != nil {
		a.log.Warn("could not write data to file", slog.String("fileName", fileName), slog.String("error", err.Error()))
		res.Status = StatusInternalServerError
		res.WriteStr("Could not write data to file: " + err.Error())
		return
	}
	w.Flush()

	a.log.Info("written file contents", slog.String("fileName", fileName), slog.Int64("bytes", n))

	res.Status = StatusCreated
}
