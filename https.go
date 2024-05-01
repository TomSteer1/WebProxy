package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"
)

func handleHTTPS(w http.ResponseWriter, r *http.Request) {
	Info.Println("HTTPS request received")

	var req *ProxyRequest = &ProxyRequest{Request: r, Writer: w, Secure: true, Handled: false}
	// Add to queue
	queueRequest(req)

	for !req.Handled {
		// Wait until the request is handled
		time.Sleep(1 * time.Second)
	}
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	handlePass(tr, w, r)
}

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	// Change host to be a locally controlled host
	var originalHost string = r.Host
	var newHost string = "localhost:8080"
	Debug.Printf("Tunneling request to %s from %s ", originalHost, r.RemoteAddr)
	dest_conn, err := net.DialTimeout("tcp", newHost, 10*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func startHttpsServer() *http.Server {
	Info.Println("Starting HTTPS server")
	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Scheme = "https"
			r.URL.Host = r.Host
			handleHTTPS(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	go server.ListenAndServeTLS(config.SSLCert, config.SSLKey)
	return server
}

func handlePass(tr *http.Transport, w http.ResponseWriter, r *http.Request) {
	resp, err := tr.RoundTrip(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		handleError(err, "Error in handleHTTPsPass", false)
		return
	}
	defer resp.Body.Close()
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
