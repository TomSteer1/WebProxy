package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
)

var clients = make(map[*websocket.Conn]bool)

func startWebSocketServer() {
	Info.Println("Starting WebSocket server")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// http.ServeFile(w, r, "web/"+r.URL.Path[1:])
		handler := http.FileServerFS(publicFs)
		r.URL.Path = "web/" + r.URL.Path
		handler.ServeHTTP(w, r)

	})
	http.HandleFunc("/ca.crt", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, secretFs, "certs/ca.crt")
	})
	http.HandleFunc("/ws", handleWebSocket)
	Error.Fatal(http.ListenAndServe(":8000", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Error.Println("Failed to upgrade connection to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Add client to clients map
	clients[conn] = true

	// Read and write messages
	for {
		// Read message from client
		_, msg, err := conn.ReadMessage()
		if err != nil {
			Error.Println("Failed to read message from client:", err)
			break
		}

		message := strings.TrimSpace(string(msg))
		// Marshal message
		var request SocketRequest
		var response SocketRequest
		err = json.Unmarshal([]byte(message), &request)
		if err != nil {
			Error.Println("Failed to unmarshal message:", err)
			response = newResponseError("Failed to unmarshal message", err)
			msg, err = json.Marshal(response)
		} else {
			// Print received message
			Debug.Println("Received message:", string(request.Action))
			switch string(request.Action) {
			case "ping":
				response.Msg = "pong"
				response.Msgtype = "ping"
				msg, err = json.Marshal(response)
			case "get_req_queue":
				getReqQueue(&msg)
			case "get_resp_queue":
				getRespQueue(&msg)
			case "pass_req":
				passRequest(&request, &msg)
			case "pass_resp":
				passResponse(&request, &msg)
			case "drop":
				dropRequest(&request, &msg)
			case "get_history":
				response.Queue = make([]QueueItem, 0)
				for _, request := range history {
					Debug.Println("Host:", request.Request.Host)
					response.Queue = append(response.Queue, convertProxyToHistoryQueue(request))
				}
				response.Msgtype = "history"
				msg, err = json.Marshal(response)
			case "get_settings":
				response.Msgtype = "settings"
				response.Settings = settings
				msg, err = json.Marshal(response)
			case "set_settings":
				settings = request.Settings
				Debug.Println("Settings:", settings)
				response.Msgtype = "settings"
				response.Settings = settings
				msg, err = json.Marshal(response)
				broadcastMessage(string(msg))
			default:
				response = newResponseError("Unknown action", nil)
				msg, err = json.Marshal(response)
			}
		}
		if err != nil {
			Error.Println("Failed to marshal message:", err)
			break
		}

		// Write message back to client
		err = conn.WriteMessage(websocket.TextMessage, msg)
		if err != nil {
			Error.Println("Failed to write message to client:", err)
			break
		}
	}
}

func broadcastMessage(message string) {
	for client := range clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			Error.Println("Failed to broadcast message to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func queueRequest(proxyRequest *ProxyRequest) {
	reqQueue <- proxyRequest
	addToHistory(proxyRequest)
	var socketMessage SocketRequest
	socketMessage.Msgtype = "newRequest"
	socketMessage.Queue = make([]QueueItem, 1)
	socketMessage.Queue[0] = convertProxyToReqQueue(proxyRequest)
	msg, _ := json.Marshal(socketMessage)
	broadcastMessage(string(msg))
}

func queueResponse(proxyRequest *ProxyRequest) {
	proxyRequest.Handled = false
	respQueue <- proxyRequest
	var socketMessage SocketRequest
	socketMessage.Msgtype = "newResponse"
	socketMessage.Queue = make([]QueueItem, 1)
	socketMessage.Queue[0] = convertProxyToRespQueue(proxyRequest)
	msg, _ := json.Marshal(socketMessage)
	broadcastMessage(string(msg))
}

func addToHistory(proxyRequest *ProxyRequest) {
	history[proxyRequest.UUID] = proxyRequest
	Info.Println("Host:", proxyRequest.Request.Host)
}

func convertProxyToReqQueue(proxyRequest *ProxyRequest) QueueItem {
	var queueItem QueueItem
	queueItem.Method = proxyRequest.Request.Method
	queueItem.Path = proxyRequest.Request.URL.Path
	queueItem.Headers = proxyRequest.Request.Header
	queueItem.UUID = proxyRequest.UUID
	queueItem.Host = proxyRequest.Request.Host
	queueItem.Query = proxyRequest.Request.URL.RawQuery
	queueItem.Cookies = proxyRequest.Request.Cookies()

	var body []byte
	if proxyRequest.Request.Body != nil {
		body, _ = io.ReadAll(proxyRequest.Request.Body)
		queueItem.Body = string(body)
	}

	proxyRequest.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	return queueItem
}

func convertProxyToRespQueue(proxyRequest *ProxyRequest) QueueItem {
	var queueItem QueueItem
	queueItem.Status = proxyRequest.Response.StatusCode
	queueItem.Headers = proxyRequest.Response.Header
	queueItem.UUID = proxyRequest.UUID
	queueItem.Cookies = proxyRequest.Response.Cookies()
	var body []byte
	if proxyRequest.Response.Body != nil {
		body, _ = io.ReadAll(proxyRequest.Response.Body)
		queueItem.Body = string(body)
	}
	proxyRequest.Response.Body = io.NopCloser(strings.NewReader(string(body)))
	return queueItem
}

func convertProxyToHistoryQueue(proxyRequest *ProxyRequest) QueueItem {
	var queueItem QueueItem
	queueItem.Method = proxyRequest.Request.Method
	queueItem.Path = proxyRequest.Request.URL.Path
	queueItem.Headers = proxyRequest.Request.Header
	queueItem.UUID = proxyRequest.UUID
	queueItem.Host = proxyRequest.Request.Host
	queueItem.Query = proxyRequest.Request.URL.RawQuery
	queueItem.Cookies = proxyRequest.Request.Cookies()

	var body []byte
	if proxyRequest.Request.Body != nil {
		body, _ = io.ReadAll(proxyRequest.Request.Body)
		queueItem.Body = string(body)
		proxyRequest.Request.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	if proxyRequest.Response != nil {
		queueItem.Status = proxyRequest.Response.StatusCode
		queueItem.StatusMessage = proxyRequest.Response.Status
		queueItem.RespHeaders = proxyRequest.Response.Header
		if proxyRequest.Response.Body != nil {
			body, _ = io.ReadAll(proxyRequest.Response.Body)
			queueItem.RespBody = string(body)
			proxyRequest.Response.Body = io.NopCloser(strings.NewReader(string(body)))
		}
	}

	queueItem.TimeStamp = proxyRequest.TimeStamp
	return queueItem
}

func newResponseError(message string, err error) SocketRequest {
	var response SocketRequest
	response.Error = err
	response.Msgtype = "error"
	response.Msg = message
	return response
}

func getReqQueue(msg *[]byte) {
	var resp SocketRequest
	// Get queue but do not pop
	resp.Queue = make([]QueueItem, 0)
	for i := 0; i < len(reqQueue); i++ {
		request := <-reqQueue
		resp.Queue = append(resp.Queue, convertProxyToReqQueue(request))
		reqQueue <- request
	}
	resp.Msgtype = "req_queue"
	*msg, resp.Error = json.Marshal(resp)
}

func getRespQueue(msg *[]byte) {
	var resp SocketRequest
	// Get queue but do not pop
	resp.Queue = make([]QueueItem, 0)
	for i := 0; i < len(respQueue); i++ {
		request := <-respQueue
		resp.Queue = append(resp.Queue, convertProxyToRespQueue(request))
		respQueue <- request
	}
	resp.Msgtype = "resp_queue"
	*msg, resp.Error = json.Marshal(resp)
}

func passRequest(req *SocketRequest, msg *[]byte) {
	var response SocketRequest
	if len(reqQueue) == 0 {
		response = newResponseError("Queue is empty", nil)
		*msg, response.Error = json.Marshal(response)
		return
	}
	if !passUUID(req.UUID, req.Queue[0]) {
		response = newResponseError("Request UUID does not match", nil)
	} else {
		response.Msgtype = "success"
	}
	*msg, response.Error = json.Marshal(response)
}

func passResponse(req *SocketRequest, msg *[]byte) {
	var response SocketRequest
	if len(respQueue) == 0 {
		response = newResponseError("Queue is empty", nil)
		*msg, response.Error = json.Marshal(response)
		return
	}
	if !passRespUUID(req.UUID, req.Queue[0]) {
		response = newResponseError("Response UUID does not match", nil)
	} else {
		response.Msgtype = "success"
	}
	*msg, response.Error = json.Marshal(response)
}

func passRespUUID(uuid string, newItem ...QueueItem) bool {
	if len(respQueue) == 0 {
		return false
	}
	response := <-respQueue
	if response.UUID != uuid {
		// Add back to start of queue
		respQueue <- response
		// Rotate queue
		for i := 0; i < len(respQueue)-1; i++ {
			respQueue <- <-respQueue
		}
		return false
	}
	if len(newItem) == 0 {
		response.Handled = true
	} else {
		passRequest := newItem[0]
		response.Response.StatusCode = passRequest.Status
		response.Response.Header = passRequest.Headers
		response.Response.Body = io.NopCloser(strings.NewReader(passRequest.Body))
		response.Response.Header.Set("Content-Length", strconv.FormatInt(int64(len(passRequest.Body)), 10))
		response.Response.ContentLength = int64(len(passRequest.Body))
		for _, cookie := range passRequest.Cookies {
			response.Response.Header.Add("Set-Cookie", cookie.String())
		}
		response.Handled = true
	}
	var broadcast SocketRequest
	broadcast.Msgtype = "handled"
	broadcast.UUID = response.UUID
	msg, _ := json.Marshal(broadcast)
	broadcastMessage(string(msg))
	return true
}

func passUUID(uuid string, newItem ...QueueItem) bool {
	if len(reqQueue) == 0 {
		return false
	}
	request := <-reqQueue
	if request.UUID != uuid {
		// Add back to start of queue
		reqQueue <- request
		// Rotate queue
		for i := 0; i < len(reqQueue)-1; i++ {
			reqQueue <- <-reqQueue
		}
		return false
	}
	if len(newItem) == 0 {
		request.Handled = true
	} else {
		passRequest := newItem[0]
		request.Request.Method = passRequest.Method
		request.Request.URL.Path = passRequest.Path
		if passRequest.Host != "" {
			request.Request.Host = passRequest.Host
		}
		request.Request.Header = passRequest.Headers
		request.Request.Body = io.NopCloser(strings.NewReader(passRequest.Body))
		// UrlDecode query
		decodedQuery, err := url.QueryUnescape(passRequest.Query)
		handleError(err, "Error in passUUID", false)
		request.Request.URL.RawQuery = decodedQuery
		request.Request.Header.Set("Content-Length", strconv.FormatInt(int64(len(passRequest.Body)), 10))
		request.Request.ContentLength = int64(len(passRequest.Body))
		for _, cookie := range passRequest.Cookies {
			request.Request.AddCookie(cookie)
		}
		request.Handled = true
	}
	var broadcast SocketRequest
	broadcast.Msgtype = "handled"
	broadcast.UUID = request.UUID
	msg, _ := json.Marshal(broadcast)
	broadcastMessage(string(msg))
	return true
}

func dropRequest(req *SocketRequest, msg *[]byte) {
	var response SocketRequest
	if !dropUUID(req.UUID) {
		response = newResponseError("UUID does not match", nil)
	} else {
		response.Msgtype = "dropped"
	}
	*msg, response.Error = json.Marshal(response)
}

func dropUUID(uuid string) bool {
	if len(reqQueue) == 0 {
		return false
	}
	request := <-reqQueue
	if request.UUID != uuid {
		// Add back to start of queue
		reqQueue <- request
		// Rotate queue
		for i := 0; i < len(reqQueue)-1; i++ {
			reqQueue <- <-reqQueue
		}
		return false
	}
	request.Dropped = true
	request.Handled = true
	var broadcast SocketRequest
	broadcast.Msgtype = "handled"
	broadcast.UUID = request.UUID
	msg, _ := json.Marshal(broadcast)
	broadcastMessage(string(msg))
	return true
}
