package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"slices"
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

func main() {
	fmt.Println("Logs from your program will appear here!")
	port := 4221

	cfg := parseConfig()
	fmt.Printf("Parsed config: %s\n", cfg.Debug())

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", port)
		os.Exit(1)
	}

	fmt.Printf("Started server on port :%d\n", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		go handleConn(cfg, conn)
	}
}

func handleConn(cfg Config, stream io.ReadWriteCloser) {
	defer stream.Close()

	req, err := Read(stream)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}

	res := newCleanResponse()
	if err := Handle(cfg, res, req); err != nil {
		fmt.Println("Error handling request: ", err.Error())
		return
	}

	acceptEncoding := parseAcceptEncodings(req.Headers[HeaderAcceptEncoding])
	if slices.Contains(acceptEncoding, EncodingGzip) {
		res.Headers[HeaderContentEncoding] = EncodingGzip
	}

	n, err := Write(stream, res)
	if err != nil {
		fmt.Printf("Failed to write response: %v (bytes written %d)\n", err, n)
		return
	}

	fmt.Printf("Handled request: %s %s (bytes written %d)\n", req.Method, req.Target, n)
}

func Handle(cfg Config, res *HttpResponse, req *HttpRequest) error {
	if req.Target == "/" {
		rootHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/echo/") {
		echoHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/user-agent") {
		userAgentHandler(res, req)
	} else if strings.HasPrefix(req.Target, "/files/") {
		filesHandler(cfg, res, req)
	} else {
		res.Status = StatusNotFound
	}
	return nil
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
