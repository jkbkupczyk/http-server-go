package main

import (
	"log/slog"
	"net"
	"slices"
	"strings"
)

type Handler func(*HttpRequest, *HttpResponse)

type Server struct {
	Addr     string
	Handler  Handler
	log      *slog.Logger
	listener net.Listener
}

func NewServerFromConfig(addr string, logger *slog.Logger, handler Handler) (*Server, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if handler == nil {
		handler = defaultHandler()
	}

	return &Server{
		Addr:    addr,
		Handler: handler,
		log:     logger,
	}, nil
}

func defaultHandler() Handler {
	return func(req *HttpRequest, res *HttpResponse) {
		res.Status = StatusOK
	}
}

func (srv *Server) Start() error {
	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return err
	}
	defer func() {
		if err := l.Close(); err != nil {
			srv.log.Error("could not close server", slog.String("error", err.Error()))
		}
	}()
	srv.listener = l

	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			srv.log.Warn("could not accept connection", slog.String("error", err.Error()))
			continue
		}

		go srv.handleConn(conn)
	}
}

func (srv *Server) handleConn(conn net.Conn) {
	defer func() {
		if err := conn.Close(); err != nil {
			srv.log.Warn("could not close connection", slog.String("error", err.Error()))
		}
	}()

	for {
		req, err := Read(conn)
		if err != nil {
			srv.log.Error("could not read request", slog.String("error", err.Error()))
			break
		}

		res := newCleanResponse()
		srv.Handler(req, res)

		acceptEncoding := parseAcceptEncodings(req.Headers[HeaderAcceptEncoding])
		if slices.Contains(acceptEncoding, EncodingGzip) {
			res.Headers[HeaderContentEncoding] = EncodingGzip
		}

		var closeConnection bool
		if strings.ToLower(req.Headers[HeaderConnection]) == "close" {
			closeConnection = true
			res.Headers[HeaderConnection] = "close"
		}

		n, err := Write(conn, res)
		if err != nil {
			srv.log.Error("could not write request", slog.String("error", err.Error()))
			break
		}

		srv.log.Info("handled request",
			slog.String("method", req.Method),
			slog.String("target", req.Target),
			slog.Int64("bytes", n),
		)

		if closeConnection {
			break
		}
	}
}

func parseAcceptEncodings(acceptEncHeader string) []string {
	if acceptEncHeader == "" {
		return []string{}
	}

	encodings := make([]string, 0)
	for _, v := range strings.Split(acceptEncHeader, ",") {
		encodings = append(encodings, strings.ToLower(strings.TrimSpace(v)))
	}

	return encodings
}
