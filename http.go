package main

// Old functions from when http and https was split

// func handleHTTP(w http.ResponseWriter, r *http.Request) {
// 	Info.Println("HTTP request received")
// 	var req *ProxyRequest = &ProxyRequest{Request: r, Writer: w, Secure: true, Handled: false, TimeStamp: time.Now().UnixMilli()}
// 	req.UUID = generateUUID()
// 	if settings.Enabled && checkRules(req) {
// 		// Add to queue
// 		queueRequest(req)

// 		for !req.Handled && settings.Enabled {
// 			// Wait until the request is handled
// 			time.Sleep(1 * time.Second)
// 		}
// 		if req.Dropped {
// 			return
// 		}

// 		if !req.Handled {
// 			passUUID(req.UUID)
// 		}

// 	} else {
// 		addToHistory(req)
// 	}
// 	tr := &http.Transport{}
// 	handlePass(tr, req)
// }

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
