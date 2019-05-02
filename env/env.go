package env

import (
	"fmt"
	"log"
	"os"
)

func GetOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func GetOrError(key string) (string, error) {
	if v, ok := os.LookupEnv(key); ok {
		return v, nil
	}
	return "", fmt.Errorf("env %s must be sed but not found", key)
}

func Get(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	log.Fatalf("Env %s must be sed but not found.", key)
	return ""
}
