package main

import (
	"io"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	SSLKey          string
	SSLCert         string
	ProxyListenPort int
	SSLListenPort   int
	DebugMode       bool
}

func NewConfig() *Config {
	config := new(Config)

	// Load enviroment file
	err := godotenv.Load(".env")
	if err != nil {
		Error.Fatal("Error loading .env file")
	}

	loadEnv(&config.DebugMode, "DebugMode", false)
	if !config.DebugMode {
		Debug.SetOutput(io.Discard)
	}

	loadEnv(&config.SSLKey, "SSLKey", "server.key")
	loadEnv(&config.SSLCert, "SSLCert", "server.crt")
	loadEnv(&config.SSLListenPort, "SSLListenPort", 8080)
	loadEnv(&config.ProxyListenPort, "ProxyListenPort", 8888)
	return config
}

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
