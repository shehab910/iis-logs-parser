package config

import "os"

const (
	TOKEN_EXPIRATION_MINS = 60
)

const (
	SERVER_PORT_DEFAULT = "8080"
)

func GetServerPortOrDefault() string {
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = SERVER_PORT_DEFAULT
	}
	return serverPort
}
