package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

type SocketRequest struct {
	Action  string      `json:"action,omitempty"`
	Msgtype string      `json:"msgtype,omitempty"`
	Msg     string      `json:"msg,omitempty"`
	Queue   []QueueItem `json:"queue"`
	Error   error       `json:"error,omitempty"`
}

type QueueItem struct {
	Method  string      `json:"method"`
	URL     string      `json:"url"`
	Body    string      `json:"body"`
	Headers http.Header `json:"headers"`
	UUID    string      `json:"uuid"`
}

var clients = make(map[*websocket.Conn]bool)

func startWebSocketServer() {
	Info.Println("Starting WebSocket server")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/"+r.URL.Path[1:])
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
			response.Error = err
			response.Msgtype = "error"
			response.Msg = "Failed to unmarshal message"
			msg, err = json.Marshal(response)
		} else {
			// Print received message
			Debug.Println("Received message:", string(request.Action))
			switch string(request.Action) {
			case "ping":
				response.Msg = "pong"
				response.Msgtype = "ping"
				msg, err = json.Marshal(response)
			case "get_queue":
				// Get queue but do not pop
				response.Queue = make([]QueueItem, 0)
				Info.Println("Queue length:", len(queue))
				for i := 0; i < len(queue); i++ {
					request := <-queue
					var item QueueItem
					item.Method = request.Request.Method
					item.URL = request.Request.URL.String()
					item.Headers = request.Request.Header
					var body []byte
					if request.Request.Body != nil {
						body, err = io.ReadAll(request.Request.Body)
						if err != nil {
							Error.Println("Failed to read request body:", err)
							break
						}
					}
					item.Body = string(body)

					response.Queue = append(response.Queue, item)
					queue <- request
				}
				response.Msgtype = "queue"
				msg, err = json.Marshal(response)
			case "pop":
				// Pop queue if not empty
				if len(queue) == 0 {
					response.Error = err
					response.Msgtype = "error"
					response.Msg = "Queue is empty"
					msg, err = json.Marshal(response)
					break
				}
				request := <-queue
				request.Handled = true
				response.Msgtype = "pop"
				response.Msg = "Request popped"
				msg, err = json.Marshal(response)
			default:
				response.Error = err
				response.Msgtype = "error"
				response.Msg = "Invalid action"
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
	queue <- proxyRequest
	var socketMessage SocketRequest
	var queueItem QueueItem
	socketMessage.Msgtype = "newRequest"
	socketMessage.Queue = make([]QueueItem, 1)
	queueItem.Method = proxyRequest.Request.Method
	queueItem.URL = proxyRequest.Request.URL.String()
	queueItem.Headers = proxyRequest.Request.Header
	var body []byte
	if proxyRequest.Request.Body != nil {
		body, _ = io.ReadAll(proxyRequest.Request.Body)
		queueItem.Body = string(body)
	}
	socketMessage.Queue[0] = queueItem
	msg, _ := json.Marshal(socketMessage)
	broadcastMessage(string(msg))
}
