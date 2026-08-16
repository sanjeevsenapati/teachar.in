package config

import "os"

// Config holds the application configuration.
type Config struct {
	AppName string
	Host    string
	Port    string
	Env     string
}

// New creates a new Config instance from environment variables with defaults.
// Using environment variables for configuration is a standard practice in
// modern applications (following the 12-factor app methodology).
func New() (*Config, error) {
	appName := os.Getenv("APP_NAME")
	if appName == "" {
		appName = "teachar.in"
	}
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	return &Config{AppName: appName, Host: host, Port: port, Env: env}, nil
}
