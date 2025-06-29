package main

import (
	"bufio"
	"io"
)

func newBufferedWriter(w io.Writer) *bufio.Writer {
	buffWriter, ok := w.(*bufio.Writer)
	if ok {
		return buffWriter
	} else {
		return bufio.NewWriter(w)
	}
}

func newBufferedReader(r io.Reader) *bufio.Reader {
	buffReader, ok := r.(*bufio.Reader)
	if ok {
		return buffReader
	} else {
		return bufio.NewReader(r)
	}
}
