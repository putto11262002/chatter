package chatter

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type ConfigLoader interface {
	Load() (*Config, error)
}

// EnvConfigLoader loads the configuration from environment variables.
// The SECRET environment variable is expected to be a base64-encoded string.
// It is decoded into a byte slice and used as the secret key for signing JWT tokens.
// The ALLOWED_ORIGINS environment variable is expected to be a comma-separated list
// of origins that are allowed to connect to the server.
type EnvConfigLoader struct {
}

func (l *EnvConfigLoader) Load() (*Config, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	// Parse values from environment variables
	port, _ := strconv.Atoi(getEnv("PORT"))

	secret, err := base64.StdEncoding.DecodeString(getEnv("SECRET"))
	if err != nil {
		return nil, errors.New("invalid secret value")
	}

	allowedOrigins := strings.Split(getEnv("ALLOWED_ORIGINS"), ",")

	return &Config{
		Port:           port,
		Hostname:       getEnv("HOSTNAME"),
		Secret:         secret,
		SQLiteFile:     getEnv("SQLITE_FILE"),
		MigrationDir:   getEnv("MIGRATION_DIR"),
		AllowedOrigins: allowedOrigins,
	}, nil
}

type DefaultConfigLoader struct {
}

func (l *DefaultConfigLoader) Load() (*Config, error) {
	// Generate a random secret
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, errors.New("failed to generate secret")
	}

	return &Config{
		Port:           8080,
		Hostname:       "0.0.0.0",
		Secret:         secret,
		AllowedOrigins: []string{"*"},
	}, nil
}

// Utility function to get an environment variable with a default value
func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return ""
	}
	return value
}
