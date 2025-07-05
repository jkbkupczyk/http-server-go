package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
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

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			continue
		}
		go handleConn(cfg, conn)
	}
}

func handleConn(cfg Config, conn net.Conn) {
	defer conn.Close()

	req, err := Read(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}

	res, err := Handle(cfg, newCleanResponse(), req)
	if err != nil {
		return
	}

	n, err := Write(conn, res)
	if err != nil {
		fmt.Printf("Failed to write response: %v (bytes written %d)\n", err, n)
		return
	}

	return
}

func Handle(cfg Config, res *HttpResponse, req *HttpRequest) (*HttpResponse, error) {
	if req.Target == "/" {
		res.Status = 200
	} else if strings.HasPrefix(req.Target, "/echo/") {
		res.Status = 200
		echo, _ := strings.CutPrefix(req.Target, "/echo/")
		res.Body = strings.NewReader(echo)
		res.BodyLength = func() int { return len(echo) }
		res.Headers["Content-Type"] = "text/plain"
	} else if strings.HasPrefix(req.Target, "/user-agent") {
		res.Status = 200
		body := req.Headers["User-Agent"]
		res.Body = strings.NewReader(body)
		res.BodyLength = func() int { return len(body) }
		res.Headers["Content-Type"] = "text/plain"
	} else if strings.HasPrefix(req.Target, "/files/") {
		res.Status = 200
		fileName, _ := strings.CutPrefix(req.Target, "/files/")
		f, err := os.Open(filepath.Join(cfg.FileDir, fileName))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				res.Status = 404
				return res, nil
			} else {
				resp := fmt.Sprintf("Could not load file: %s", err.Error())
				res.Body = strings.NewReader(resp)
				res.BodyLength = func() int { return len(resp) }
				res.Headers["Content-Type"] = "text/plain"
			}
		} else {
			res.Body = f
			res.BodyLength = func() int {
				stat, err := f.Stat()
				if err != nil {
					return 0
				}
				return int(stat.Size())
			}
			res.Headers["Content-Type"] = "application/octet-stream"
		}
	} else {
		res.Status = 404
	}
	return res, nil
}
