package main

import (
	"fmt"
	"io"
)

type HttpHeaders map[string]string

type HttpResponse struct {
	Version string
	Status  int
	Headers HttpHeaders
	Body    io.Reader
}

func Write(w io.Writer, res HttpResponse) (int64, error) {
	bw := newBufferedWriter(w)
	total := int64(0)

	// Status line
	n, err := fmt.Fprintf(bw, "%s %d %s\r\n", res.Version, res.Status, statusString(res.Status))
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Headers
	if res.Headers != nil {
		for k, v := range res.Headers {
			nn, err := fmt.Fprintf(bw, "%s: %s\r\n", k, v)
			total += int64(nn)
			if err != nil {
				return total, err
			}
		}
	}

	n, err = bw.WriteString("\r\n")
	total += int64(n)
	if err != nil {
		return total, err
	}

	// Body
	if res.Body != nil {
		nn, err := io.Copy(bw, res.Body)
		total += nn
		if err != nil {
			return total, err
		}
	}

	return total, bw.Flush()
}

func statusString(code int) string {
	switch code {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}
