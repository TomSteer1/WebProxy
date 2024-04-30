package main

import (
	"log"
	"os"
)

var (
	Info  log.Logger
	Debug log.Logger
	Error log.Logger
)

func initLoggers() {
	Info = *log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = *log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = *log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
