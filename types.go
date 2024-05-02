package main

import "net/http"

type SocketRequest struct {
	Action   string      `json:"action,omitempty"`
	Msgtype  string      `json:"msgtype,omitempty"`
	Msg      string      `json:"msg,omitempty"`
	Queue    []QueueItem `json:"queue"`
	Error    error       `json:"error,omitempty"`
	UUID     string      `json:"uuid,omitempty"`
	Settings Settings    `json:"settings,omitempty"`
}

type QueueItem struct {
	Method        string         `json:"method"`
	Path          string         `json:"path"`
	Body          string         `json:"body"`
	Query         string         `json:"query"`
	Headers       http.Header    `json:"headers"`
	Cookies       []*http.Cookie `json:"cookies"`
	UUID          string         `json:"uuid"`
	Host          string         `json:"host"`
	Status        int            `json:"status"`
	StatusMessage string         `json:"statusMessage"`
	RespBody      string         `json:"respBody"`    // Only used for history
	RespHeaders   http.Header    `json:"respHeaders"` // Only used for history
	TimeStamp     int64          `json:"timestamp"`   // Only used for history
}

type Settings struct {
	Enabled       bool     `json:"enabled"`
	IgnoredTypes  []string `json:"ignoredTypes"`
	ProxyPort     int      `json:"proxyPort"`
	CatchResponse bool     `json:"catchResponse"`
	IgnoredHosts  []string `json:"ignoredHosts"`
	Whitelist     bool     `json:"whitelist"`
	Regex         bool     `json:"useRegex"`
}

type ProxyRequest struct {
	Request   *http.Request
	Writer    http.ResponseWriter
	Response  *http.Response
	Secure    bool
	Handled   bool
	Dropped   bool
	UUID      string
	TimeStamp int64
}

type Config struct {
	SSLKey             string
	SSLCert            string
	ProxyListenPort    int
	SSLListenPort      int
	DebugMode          bool
	ProxyListenAddress string
}
