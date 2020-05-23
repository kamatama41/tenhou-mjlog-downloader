package main

import (
	"fmt"
	"log"
	"net/http"

	auth "github.com/abbot/go-http-auth"
	"github.com/kamatama41/tenhou-mjlog-downloader/crawler"
	"github.com/kamatama41/tenhou-mjlog-downloader/env"
	"github.com/kamatama41/tenhou-mjlog-downloader/twitter"
	"golang.org/x/crypto/bcrypt"
)

var (
	basicUser                string
	basicPass                string
	port                     string
	userName                 string
	storageType              string
	twitterConsumerKey       string
	twitterConsumerSecret    string
	twitterAccessToken       string
	twitterAccessTokenSecret string
)

func init() {
	basicUser = env.Get("BASIC_AUTH_USER")
	basicPass = env.Get("BASIC_AUTH_PASSWORD")
	port = env.GetOrDefault("PORT", "8080")
	userName = env.Get("TENHOU_USER_NAME")
	storageType = env.GetOrDefault("STORAGE_TYPE", "local")
	twitterConsumerKey = env.GetOrDefault("TWITTER_CONSUMER_KEY", "")
	twitterConsumerSecret = env.GetOrDefault("TWITTER_CONSUMER_SECRET", "")
	twitterAccessToken = env.GetOrDefault("TWITTER_ACCESS_TOKEN", "")
	twitterAccessTokenSecret = env.GetOrDefault("TWITTER_ACCESS_TOKEN_SECRET", "")
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
			twitterCli := twitter.New(twitterConsumerKey, twitterConsumerSecret, twitterAccessToken, twitterAccessTokenSecret)
			c, err := crawler.New(userName, storageType, twitterCli)
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

	log.Printf("Starting server on :%s", port)
	authenticator := auth.NewBasicAuthenticator("", secret)
	http.HandleFunc("/crawl", authenticator.Wrap(crawlHandler))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
