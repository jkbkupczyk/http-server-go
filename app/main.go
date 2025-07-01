package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	conn, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	req, err := Read(conn)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}

	res := HttpResponse{
		Version: "HTTP/1.1",
		Status:  404,
		Headers: HttpHeaders{},
	}

	if req.Target == "/" {
		res.Status = 200
	} else {
		res.Status = 404
	}

	n, err := Write(conn, res)
	if err != nil {
		fmt.Printf("Failed to write response: %v (bytes written %d)\n", err, n)
		os.Exit(1)
	}

}
