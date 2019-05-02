package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	auth "github.com/abbot/go-http-auth"

	"golang.org/x/crypto/bcrypt"
)

var (
	basicUser string
	basicPass string
	port      string
)

func init() {
	basicUser = getEnv("BASIC_AUTH_USER")
	basicPass = getEnv("BASIC_AUTH_PASSWORD")
	port = getEnvOrDefault("PORT", "8080")
}

func getEnvOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

func getEnv(key string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	log.Fatalf("Env %s must be sed but not found.", key)
	return ""
}

func secret(user, realm string) string {
	if user == basicUser {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(basicPass), bcrypt.DefaultCost)
		if err == nil {
			return string(hashedPassword)
		}
	}
	return ""
}

func main() {
	crawlHandler := func(w http.ResponseWriter, req *auth.AuthenticatedRequest) {
		switch req.Method {
		case "POST":
			err := crawl()
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			} else {
				fmt.Fprint(w, "OK")
			}
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}

	log.Println("Starting server..")
	authenticator := auth.NewBasicAuthenticator("", secret)
	http.HandleFunc("/crawl", authenticator.Wrap(crawlHandler))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
