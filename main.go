package main

import (
	"fmt"
	"log"
	"net/http"

	auth "github.com/abbot/go-http-auth"
	"github.com/kamatama41/tenhou-mjlog-downloader/crawler"
	"github.com/kamatama41/tenhou-mjlog-downloader/env"
	"golang.org/x/crypto/bcrypt"
)

var (
	basicUser   string
	basicPass   string
	port        string
	userName    string
	storageType string
)

func init() {
	basicUser = env.Get("BASIC_AUTH_USER")
	basicPass = env.Get("BASIC_AUTH_PASSWORD")
	port = env.GetOrDefault("PORT", "8080")
	userName = env.Get("TENHOU_USER_NAME")
	storageType = env.GetOrDefault("STORAGE_TYPE", "local")
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
			c, err := crawler.New(userName, storageType)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			err = c.Crawl()
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			fmt.Fprint(w, "OK")
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}

	log.Println("Starting server..")
	authenticator := auth.NewBasicAuthenticator("", secret)
	http.HandleFunc("/crawl", authenticator.Wrap(crawlHandler))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
