package main

import (
	"crypto/tls"
	"net/http"
)

type ProxyRequest struct {
	Request *http.Request
	Writer  http.ResponseWriter
	Secure  bool
	Handled bool
	UUID    string
}

var queue = make(chan *ProxyRequest, 100)

func startProxy() {
	server := &http.Server{
		Addr: ":8888",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleHTTP(w, r)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	Info.Println("Starting Logger")
	Error.Fatal(server.ListenAndServe())
}
