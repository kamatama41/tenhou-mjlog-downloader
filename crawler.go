package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	regDownloadLink *regexp.Regexp
	userName        string
	storageType     string
)

type result struct {
	f string
	e error
}

func init() {
	regDownloadLink = regexp.MustCompile("<a href=\"/0/log/find\\.cgi\\?log=(.+?)\">DOWNLOAD</a>")
	userName = getEnv("TENHOU_USER_NAME")
	storageType = getEnvOrDefault("STORAGE_TYPE", "local")
}

func crawl() error {
	log.Print("Start crawling..")
	fileNames := make(chan string, 1)
	wg := sync.WaitGroup{}
	resultChan := make(chan result)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go processFiles(fileNames, resultChan, &wg)
	}

	err := getFiles(fileNames)
	if err != nil {
		return err
	}

	var errorFiles []string
	go correctErrorFiles(resultChan, &errorFiles)
	wg.Wait()
	close(resultChan)

	if len(errorFiles) != 0 {
		return fmt.Errorf("failed to process the files (%s)", strings.Join(errorFiles, ", "))
	}
	return nil
}

func getFiles(fileNames chan string) error {
	netClient := &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := netClient.Get(fmt.Sprintf("https://tenhou.net/0/log/find.cgi?un=%s", userName))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch logs: %s", response.Status)
	}

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	for _, value := range regDownloadLink.FindAllStringSubmatch(string(bytes), -1) {
		fileNames <- value[1]
	}
	close(fileNames)
	return nil
}

func processFiles(fileNames chan string, resultChan chan result, wg *sync.WaitGroup) {
	defer wg.Done()

	netClient := &http.Client{
		Timeout: time.Second * 10,
	}
	storage := newStorage(storageType)

	for fileName := range fileNames {
		path := storage.getPath(fmt.Sprintf("%s.mjlog", fileName))
		log.Printf("Start processing %s", path)
		exists, err := storage.exists(path)
		if err != nil {
			log.Print(err)
			resultChan <- result{f: fileName, e: err}
			continue
		}
		if exists {
			log.Printf("%s already exists", fileName)
		} else {
			response, err := netClient.Get(fmt.Sprintf("https://tenhou.net/0/log/find.cgi?log=%s", fileName))
			if err != nil {
				log.Print(err)
				resultChan <- result{f: fileName, e: err}
				continue
			}
			log.Println(fmt.Sprintf("Try saving %s into %s", path, storageType))
			if err := storage.save(path, response.Body); err != nil {
				log.Print(err)
				resultChan <- result{f: fileName, e: err}
				continue
			}
		}
	}
}

func correctErrorFiles(resultChan chan result, errorFiles *[]string) {
	for result := range resultChan {
		*errorFiles = append(*errorFiles, result.f)
	}
}
