package main

import (
	"crypto/tls"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var reqQueue = make(chan *ProxyRequest, 100)
var respQueue = make(chan *ProxyRequest, 100)
var history = make(map[string]*ProxyRequest)

func startProxy() {
	server := &http.Server{
		Addr: config.ProxyListenAddress + ":" + strconv.Itoa(config.ProxyListenPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				handleTunneling(w, r)
			} else {
				handleRequest(w, r)
			}
		}),
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	Info.Println("Starting Logger")
	Error.Fatal(server.ListenAndServe())
}

func checkRules(req *ProxyRequest) bool {
	// Get filetype
	ft := req.Request.URL.Path
	if strings.Contains(ft, ".") {
		ft = ft[strings.LastIndex(ft, ".")+1:]
	} else {
		ft = ""
	}
	// Check if filetype is in ignored list
	if includes(settings.IgnoredTypes, ft) {
		return false
	}
	// Check if filetype is in ignored groups
	if includes(settings.IgnoredTypes, FileCategories[ft]) {
		return false
	}
	// Check if host is in ignored list
	if settings.Whitelist {
		if settings.Regex {
			if !includesRegex(settings.IgnoredHosts, req.Request.Host) {
				return false
			}
		} else {
			if !includes(settings.IgnoredHosts, req.Request.Host) {
				return false
			}
		}
	} else {
		if settings.Regex {
			if includesRegex(settings.IgnoredHosts, req.Request.Host) {
				return false
			}
		} else {
			if includes(settings.IgnoredHosts, req.Request.Host) {
				return false
			}
		}
	}
	return true
}

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
		req.queueRequest()

		for !req.Handled && settings.Enabled {
			// Wait until the request is handled
			time.Sleep(1 * time.Second)
		}

		if !req.Handled {
			passUUID(req.UUID)
		}

	} else {
		req.addToHistory()
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
