package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

	fmt.Printf("before read\n")
	req, err := Read(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return
	}
	fmt.Printf("after read: %+v\n", req)

	res := newCleanResponse()
	if err := Handle(cfg, res, req); err != nil {
		fmt.Println("Error handling request: ", err.Error())
		return
	}

	n, err := Write(conn, res)
	if err != nil {
		fmt.Printf("Failed to write response: %v (bytes written %d)\n", err, n)
		return
	}

	return
}

func Handle(cfg Config, res *HttpResponse, req *HttpRequest) error {
	if req.Target == "/" {
		res.Status = StatusOK
	} else if strings.HasPrefix(req.Target, "/echo/") {
		res.Status = StatusOK
		echo, _ := strings.CutPrefix(req.Target, "/echo/")
		res.WriteStr(echo)
	} else if strings.HasPrefix(req.Target, "/user-agent") {
		res.Status = StatusOK
		res.WriteStr(req.Headers["User-Agent"])
	} else if strings.HasPrefix(req.Target, "/files/") {
		if req.Method == MethodPost {
			createFileHandler(cfg, res, req)
		} else {
			readFileHandler(cfg, res, req)
		}
	} else {
		res.Status = StatusNotFound
	}
	return nil
}

func createFileHandler(cfg Config, res *HttpResponse, req *HttpRequest) {
	fileName, _ := strings.CutPrefix(req.Target, "/files/")

	sb := new(strings.Builder)
	_, err := io.Copy(sb, req.Body)
	if err != nil {
		res.Status = StatusInternalServerError
		res.WriteStr("Could not read input: " + err.Error())
		return
	}

	f, err := os.Create(filepath.Join(cfg.FileDir, fileName))
	if err != nil {
		res.Status = StatusInternalServerError
		res.WriteStr("Could not create file: " + err.Error())
		return
	}
	defer f.Close()

	_, err = f.WriteString(sb.String())
	if err != nil {
		res.Status = StatusInternalServerError
		res.WriteStr("Could not write data to file: " + err.Error())
		return
	}

	res.Status = StatusCreated
}

func readFileHandler(cfg Config, res *HttpResponse, req *HttpRequest) {
	res.Status = StatusOK
	fileName, _ := strings.CutPrefix(req.Target, "/files/")
	f, err := os.Open(filepath.Join(cfg.FileDir, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			res.Status = StatusNotFound
			return
		}
		resp := fmt.Sprintf("Could not load file: %s", err.Error())
		res.WriteStr(resp)
		return
	}

	res.Body = f
	res.BodyLength = func() int64 {
		stat, err := f.Stat()
		if err != nil {
			return 0
		}
		return stat.Size()
	}
	res.Headers[HeaderContentType] = "application/octet-stream"
}
