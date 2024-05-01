package main

import (
	"io"
	"net/http"
	"os"
)

func handleError(err error, message string, fatal bool) {
	if err != nil {
		Error.Println(message)
		if fatal {
			os.Exit(1)
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}
