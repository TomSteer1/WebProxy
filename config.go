package main

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func NewConfig() *Config {
	config := new(Config)

	// Load env file
	err := godotenv.Load(".env")
	if err != nil {
		if os.IsNotExist(err) {
			Debug.Println("No .env file found")
		} else {
			Error.Fatal("Error loading .env file")
		}
	}

	loadEnv(&config.DebugMode, "DebugMode", false)
	if !config.DebugMode {
		Debug.SetOutput(io.Discard)
	}

	loadEnv(&config.SSLKey, "SSLKey", "/tmp/proxy/certs/server.key")
	loadEnv(&config.SSLCert, "SSLCert", "/tmp/proxy/certs/server.crt")
	loadEnv(&config.SSLListenPort, "SSLListenPort", 8080)
	loadEnv(&config.ProxyListenPort, "ProxyListenPort", 8888)
	loadEnv(&config.ProxyListenAddress, "ProxyListenAddress", "127.0.0.1")
	loadEnv(&config.SocketLocation, "SocketLocation", "/tmp/https.sock")
	settings.ProxyPort = config.ProxyListenPort
	return config
}

var FileCategories = map[string]string{
	"js":    "script",
	"css":   "style",
	"png":   "image",
	"ico":   "image",
	"jpg":   "image",
	"jpeg":  "image",
	"gif":   "image",
	"svg":   "image",
	"woff":  "font",
	"woff2": "font",
	"ttf":   "font",
	"otf":   "font",
	"eot":   "font",
	"html":  "html",
	"htm":   "html",
	"xml":   "data",
	"json":  "data",
	"txt":   "text",
	"csv":   "text",
	"pdf":   "file",
	"doc":   "file",
	"docx":  "file",
	"xls":   "data",
	"php":   "php",
	"asp":   "script",
}

var settings = Settings{Enabled: true, IgnoredTypes: []string{"image", "font", "style", "script"}, CatchResponse: false, Whitelist: false, Regex: true}

func loadEnv(variable interface{}, envName string, defaultValue interface{}) {
	switch v := variable.(type) {
	case *string:
		*v = getEnvString(envName, defaultValue.(string))
		Debug.Printf("Loaded string %s with value %v", envName, *v)
	case *int:
		*v = getEnvInt(envName, defaultValue.(int))
		Debug.Printf("Loaded int %s with value %v", envName, *v)
	case *bool:
		*v = getEnvBool(envName, defaultValue.(bool))
		if envName == "DebugMode" && !*v {
			return
		}
		Debug.Printf("Loaded bool %s with value %v", envName, *v)
	default:
		log.Fatalf("Unsupported variable type: %T", variable)
	}
}

func getEnvString(envName string, defaultValue string) string {
	value := os.Getenv(envName)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(envName string, defaultValue int) int {
	value := os.Getenv(envName)
	if value == "" {
		return defaultValue
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		log.Fatalf("Invalid value for environment variable %s: %s", envName, value)
	}
	return num
}

func getEnvBool(envName string, defaultValue bool) bool {
	value := os.Getenv(envName)
	if value == "" {
		return defaultValue
	}
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatalf("Invalid value for environment variable %s: %s", envName, value)
	}
	return boolValue
}
