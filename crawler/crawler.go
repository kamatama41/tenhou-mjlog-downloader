package crawler

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	fs "github.com/kamatama41/tenhou-mjlog-downloader/file_storage"
)

var (
	regDownloadLink = regexp.MustCompile("<a href=\"/0/log/find\\.cgi\\?log=(.+?)\">DOWNLOAD</a>")
)

type crawler struct {
	userName    string
	storageType string
	storage     fs.FileStorage
}

type result struct {
	f string
	e error
}

func (c *crawler) getFiles(fileNames chan string) error {
	defer close(fileNames)

	netClient := &http.Client{
		Timeout: time.Second * 10,
	}
	response, err := netClient.Get(fmt.Sprintf("https://tenhou.net/0/log/find.cgi?un=%s", c.userName))
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
	return nil
}

func (c *crawler) processFiles(fileNames chan string, resultChan chan result, wg *sync.WaitGroup) {
	defer wg.Done()

	netClient := &http.Client{
		Timeout: time.Second * 10,
	}

	for fileName := range fileNames {
		path := c.storage.GetPath(fmt.Sprintf("%s.mjlog", fileName))
		log.Printf("Start processing %s", path)
		exists, err := c.storage.Exists(path)
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
			log.Println(fmt.Sprintf("Try saving %s into %s", path, c.storageType))
			if err := c.storage.Save(path, response.Body); err != nil {
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

func (c *crawler) Crawl() error {
	log.Print("Start crawling..")
	fileNames := make(chan string, 1)
	wg := sync.WaitGroup{}
	resultChan := make(chan result)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go c.processFiles(fileNames, resultChan, &wg)
	}

	err := c.getFiles(fileNames)
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

func New(userName, storageType string) (*crawler, error) {
	c := new(crawler)
	c.userName = userName
	c.storageType = storageType
	s, err := fs.New(storageType)
	if err != nil {
		return nil, err
	}
	c.storage = s
	return c, nil
}
