package main

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"

	"github.com/google/uuid"
)

func handleError(err error, message string, fatal bool) {
	if err != nil {
		_, file, no, _ := runtime.Caller(1)
		Error.Printf("Error : %s in %s:%d\n", message, file, no)
		if fatal {
			os.Exit(1)
		}
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

func generateUUID() string {
	return uuid.New().String()
}

func includes(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func includesRegex(slice []string, regex string) bool {
	// Ensure that the regex is valid
	_, err := regexp.Compile(regex)
	if err != nil {
		return false
	}

	for _, s := range slice {
		if regexp.MustCompile(s).MatchString(regex) {
			return true
		}
	}
	return false
}
