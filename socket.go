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

func (sr SocketRequest) send(conn *websocket.Conn) bool {
	msg, err := json.Marshal(sr)
	if handleError(err, "Failed to marshal message", false) {
		return false
	}
	err = conn.WriteMessage(websocket.TextMessage, msg)
	return !handleError(err, "Failed to write message to client", false)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if handleError(err, "Failed to upgrade connection to WebSocket", false) {
		return
	}
	defer conn.Close()

	// Add client to clients map
	clients[conn] = (config.Password == "")
	if !clients[conn] {
		var response SocketRequest
		response.Msgtype = MessageTypeAuth
		response.Msg = "required"
		if !response.send(conn) {
			return
		}
	} else {
		var response SocketRequest
		response.Msgtype = MessageTypeAuth
		response.Msg = "success"
		if !response.send(conn) {
			return
		}
	}

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
		} else {
			// Print received message
			Debug.Println("Received action:", string(request.Action))
			if request.Action != "auth" && !clients[conn] {
				response.Msgtype = MessageTypeAuth
				response.Msg = "failed"
				Warning.Printf("Client %s not authenticated", conn.RemoteAddr())
			} else {
				switch string(request.Action) {
				case "auth":
					if request.Msg == config.Password {
						response.Msgtype = MessageTypeAuth
						response.Msg = "success"
						clients[conn] = true
					} else {
						response.Msgtype = MessageTypeAuth
						response.Msg = "failed"
						if !response.send(conn) {
							return
						}
					}
				case "ping":
					response.Msg = "pong"
					response.Msgtype = MessageTypePing
				case "get_req_queue":
					response = getReqQueue()
				case "get_resp_queue":
					response = getRespQueue()
				case "pass_req":
					response = request.passRequest()
				case "pass_resp":
					response = request.passResponse()
				case "drop":
					response = request.dropRequest()
				case "get_history":
					response.Queue = make([]QueueItem, 0)
					for _, request := range history {
						response.Queue = append(response.Queue, request.toHistoryQueue())
					}
					response.Msgtype = MessageTypeHistory
				case "get_settings":
					response.Msgtype = MessageTypeSettings
					response.Settings = settings
				case "set_settings":
					response = setSettings(request.Settings)
				default:
					response = newResponseError("Unknown action", nil)
				}
			}
		}
		if !response.send(conn) {
			break
		}
	}
}

func broadcastMessage(message string) {
	for client := range clients {
		if !clients[client] {
			continue
		}
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			Error.Println("Failed to broadcast message to client:", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func setSettings(newSettings Settings) SocketRequest {
	var response SocketRequest
	settings = newSettings
	Debug.Println("Settings:", settings)
	response.Msgtype = MessageTypeSettings
	response.Settings = settings
	msg, err := json.Marshal(response)
	if handleError(err, "Failed to marshal message", false) {
		return response
	}
	broadcastMessage(string(msg))
	return response
}

func (proxyRequest *ProxyRequest) queueRequest() {
	reqQueue <- proxyRequest
	proxyRequest.addToHistory()
	var socketMessage SocketRequest
	socketMessage.Msgtype = MessageTypeNewRequest
	socketMessage.Queue = make([]QueueItem, 1)
	socketMessage.Queue[0] = proxyRequest.toReqQueue()
	msg, _ := json.Marshal(socketMessage)
	broadcastMessage(string(msg))
}

func (proxyRequest *ProxyRequest) queueResponse() {
	proxyRequest.Handled = false
	respQueue <- proxyRequest
	var socketMessage SocketRequest
	socketMessage.Msgtype = MessageTypeNewResponse
	socketMessage.Queue = make([]QueueItem, 1)
	socketMessage.Queue[0] = proxyRequest.toRespQueue()
	msg, _ := json.Marshal(socketMessage)
	broadcastMessage(string(msg))
}

func (proxyRequest *ProxyRequest) addToHistory() {
	history[proxyRequest.UUID] = proxyRequest
	Info.Println("Host:", proxyRequest.Request.Host)
}

func (proxyRequest *ProxyRequest) toReqQueue() QueueItem {
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

// func convertProxyToRespQueue(proxyRequest *ProxyRequest) QueueItem {
// 	var queueItem QueueItem
// 	queueItem.Status = proxyRequest.Response.StatusCode
// 	queueItem.Headers = proxyRequest.Response.Header
// 	queueItem.UUID = proxyRequest.UUID
// 	queueItem.Cookies = proxyRequest.Response.Cookies()
// 	var body []byte
// 	if proxyRequest.Response.Body != nil {
// 		body, _ = io.ReadAll(proxyRequest.Response.Body)
// 		queueItem.Body = string(body)
// 	}
// 	proxyRequest.Response.Body = io.NopCloser(strings.NewReader(string(body)))
// 	return queueItem
// }

func (proxyRequest *ProxyRequest) toRespQueue() QueueItem {
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

func (proxyRequest *ProxyRequest) toHistoryQueue() QueueItem {
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
	response.Msgtype = MessageTypeError
	response.Msg = message
	return response
}

func getReqQueue() SocketRequest {
	var resp SocketRequest
	// Get queue but do not pop
	resp.Queue = make([]QueueItem, 0)
	for i := 0; i < len(reqQueue); i++ {
		request := <-reqQueue
		resp.Queue = append(resp.Queue, request.toReqQueue())
		reqQueue <- request
	}
	resp.Msgtype = MessageTypeReqQueue
	return resp
}

func getRespQueue() SocketRequest {
	var resp SocketRequest
	// Get queue but do not pop
	resp.Queue = make([]QueueItem, 0)
	for i := 0; i < len(respQueue); i++ {
		request := <-respQueue
		resp.Queue = append(resp.Queue, request.toRespQueue())
		respQueue <- request
	}
	resp.Msgtype = MessageTypeRespQueue
	return resp
}

func (req *SocketRequest) passRequest() SocketRequest {
	var response SocketRequest
	if len(reqQueue) == 0 {
		response = newResponseError("Queue is empty", nil)
		return response
	}
	if !passUUID(req.UUID, req.Queue[0]) {
		response = newResponseError("Request UUID does not match", nil)
	} else {
		response.Msgtype = MessageTypeSuccess
	}
	return response
}

func (req *SocketRequest) passResponse() SocketRequest {
	var response SocketRequest
	if len(respQueue) == 0 {
		response = newResponseError("Queue is empty", nil)
		return response
	}
	if !passRespUUID(req.UUID, req.Queue[0]) {
		response = newResponseError("Response UUID does not match", nil)
	} else {
		response.Msgtype = MessageTypeSuccess
	}
	return response
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
	broadcast.Msgtype = MessageTypeHandled
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
	broadcast.Msgtype = MessageTypeHandled
	broadcast.UUID = request.UUID
	msg, _ := json.Marshal(broadcast)
	broadcastMessage(string(msg))
	return true
}

func (req *SocketRequest) dropRequest() SocketRequest {
	var response SocketRequest
	if !dropUUID(req.UUID) {
		response = newResponseError("UUID does not match", nil)
	} else {
		response.Msgtype = MessageTypeDropped
	}
	return response
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
	broadcast.Msgtype = MessageTypeHandled
	broadcast.UUID = request.UUID
	msg, _ := json.Marshal(broadcast)
	broadcastMessage(string(msg))
	return true
}
