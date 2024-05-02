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
	// List all files in fs
	files, _ := publicFs.ReadDir(".")
	for _, file := range files {
		Info.Println(file.Name() + " : " + file.Type().String())
		if file.IsDir() {
			dir, _ := publicFs.ReadDir(file.Name())
			for _, d := range dir {
				Info.Println("  " + d.Name())
			}
		}
	}

	// startSocket()
	go startHttpsServer()
	go startWebSocketServer()
	startProxy()
}
