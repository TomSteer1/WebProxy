package main

import (
	"embed"
	_ "embed"
)

var (
	config *Config
)

func init() {
	config = NewConfig()
	loadSSL()
}

//go:embed web/*
var publicFs embed.FS

//go:embed certs
var secretFs embed.FS

func main() {
	go startHttpsServer()
	go startWebSocketServer()
	startProxy()
}
