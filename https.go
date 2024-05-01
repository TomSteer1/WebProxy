package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var req *ProxyRequest = &ProxyRequest{Request: r, Writer: w, Secure: false, Handled: false, TimeStamp: time.Now().UnixMilli()}
	if r.TLS != nil {
		req.Secure = true
		Debug.Println("Secure request received")
	} else {
		Debug.Println("Insecure request received")
	}
	req.UUID = generateUUID()
	if settings.Enabled && checkRules(req) {
		// Add to queue
		queueRequest(req)

		for !req.Handled && settings.Enabled {
			// Wait until the request is handled
			time.Sleep(1 * time.Second)
		}

		if !req.Handled {
			passUUID(req.UUID)
		}

	} else {
		addToHistory(req)
	}
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	if req.Secure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	} else {
		tr = &http.Transport{
			DisableCompression: true,
		}
	}
	handlePass(tr, req)
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
			handleRequest(w, r)
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	go server.ListenAndServeTLS(config.SSLCert, config.SSLKey)
	return server
}

func handlePass(tr *http.Transport, pr *ProxyRequest) {
	// Manually disable compression to avoid issues with decompression
	tr.DisableCompression = true
	if pr.Request.Header.Get("Accept-Encoding") != "" {
		pr.Request.Header.Del("Accept-Encoding")
	}
	resp, err := tr.RoundTrip(pr.Request)
	if err != nil {
		http.Error(pr.Writer, err.Error(), http.StatusServiceUnavailable)
		handleError(err, "Error in handlePass", false)
		return
	}
	// Update history
	pr.Response = resp
	if !settings.CatchResponse || !settings.Enabled || !pr.Handled {
		defer resp.Body.Close()
		copyHeader(pr.Writer.Header(), resp.Header)
		pr.Writer.WriteHeader(resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		pr.Writer.Write(body)
		pr.Response.Body = io.NopCloser(strings.NewReader(string(body)))
		return
	} else {
		defer resp.Body.Close()
		queueReply(pr)
		for !pr.Handled && settings.Enabled && settings.CatchResponse {
			time.Sleep(1 * time.Second)
		}

		if !pr.Handled {
			passRespUUID(pr.UUID)
		}

		copyHeader(pr.Writer.Header(), resp.Header)
		pr.Writer.WriteHeader(resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		pr.Writer.Write(body)
		pr.Response.Body = io.NopCloser(strings.NewReader(string(body)))
	}
}
