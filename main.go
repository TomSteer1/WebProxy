package main

var (
	config *Config
)

func init() {
	config = NewConfig()
}

func main() {
	go startHttpsServer()
	go startWebSocketServer()
	startProxy()
}
