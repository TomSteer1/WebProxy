package main

import (
	"net/http"
	"time"
)

func handleHTTP(w http.ResponseWriter, r *http.Request) {
	Info.Println("HTTP request received")
	var req *ProxyRequest = &ProxyRequest{Request: r, Writer: w, Secure: true, Handled: false}
	// Add to queue
	queueRequest(req)

	for !req.Handled {
		// Wait until the request is handled
		time.Sleep(1 * time.Second)
	}
	tr := &http.Transport{}
	handlePass(tr, w, r)
}

// func handlePass(w http.ResponseWriter, r *http.Request) {
// 	resp, err := http.DefaultTransport.RoundTrip(r)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusServiceUnavailable)
// 		handleError(err, "Error in handlePass", false)
// 		return
// 	}
// 	defer resp.Body.Close()
// 	copyHeader(w.Header(), resp.Header)
// 	w.WriteHeader(resp.StatusCode)
// 	io.Copy(w, resp.Body)
// }
