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
	Info.Println("Download the certificate from http://localhost:8000/cert.pa")
	Info.Println("Set your proxy to ", config.ProxyListenAddress, ":", config.ProxyListenPort)
	go startHttpsServer()
	go startWebSocketServer()
	startProxy()
}
