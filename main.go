package main

var (
	config *Config
)

func init() {
	initLoggers()
	config = NewConfig()
}

func main() {
	go startHttpsServer()
	go startWebSocketServer()
	startProxy()
}
