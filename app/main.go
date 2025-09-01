package main

import (
	"flag"
	"fmt"
	"log/slog"
	"strings"
)

type Config struct {
	FileDir string
}

func (c Config) Debug() string {
	return fmt.Sprintf("cfg{FileDir: %s,}", c.FileDir)
}

func parseConfig() Config {
	var cfg Config
	flag.StringVar(&cfg.FileDir, "directory", "", "Directory where the files are stored (as an absolute path)")
	flag.Parse()
	return cfg
}

type app struct {
	cfg Config
	log *slog.Logger
}

func main() {
	logger := slog.Default()
	port := 4221

	cfg := parseConfig()
	logger.Info("parsed config", slog.String("cfg", cfg.Debug()))

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	app := app{
		cfg: cfg,
		log: logger,
	}

	server, err := NewServerFromConfig(addr, logger, app.Handle)
	if err != nil {
		logger.Error("failed to create HTTP server", slog.String("err", err.Error()))
		return
	}

	logger.Info("starting server", slog.String("address", addr))
	if err := server.Start(); err != nil {
		logger.Error("could not start HTTP server", slog.String("err", err.Error()))
		return
	}
}

func (a *app) Handle(req *HttpRequest, res *HttpResponse) {
	if req.Target == "/" {
		a.rootHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/echo/") {
		a.echoHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/user-agent") {
		a.userAgentHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/files/") {
		a.filesHandler(res, req)
	} else {
		a.notFoundHandler(res, req)
	}
}
