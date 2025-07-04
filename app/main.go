package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")
	port := 4221

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
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) error {
	defer conn.Close()

	req, err := Read(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		return err
	}

	res := HttpResponse{
		Version: "HTTP/1.1",
		Status:  404,
		Headers: HttpHeaders{},
	}

	if req.Target == "/" {
		res.Status = 200
	} else if strings.HasPrefix(req.Target, "/echo/") {
		res.Status = 200
		echo, ok := strings.CutPrefix(req.Target, "/echo/")
		if !ok {
			echo = ""
		}
		res.Body = strings.NewReader(echo)
		res.BodyLength = func() int { return len(echo) }
		res.Headers["Content-Type"] = "text/plain"
	} else if strings.HasPrefix(req.Target, "/user-agent") {
		res.Status = 200
		body := req.Headers["User-Agent"]
		res.Body = strings.NewReader(body)
		res.BodyLength = func() int { return len(body) }
		res.Headers["Content-Type"] = "text/plain"
	} else {
		res.Status = 404
	}

	n, err := Write(conn, res)
	if err != nil {
		fmt.Printf("Failed to write response: %v (bytes written %d)\n", err, n)
		return err
	}

	return nil
}
