package crawler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kamatama41/tenhou-go"
	fs "github.com/kamatama41/tenhou-mjlog-downloader/file_storage"
	"github.com/kamatama41/tenhou-mjlog-downloader/twitter"
)

var (
	regDownloadLink = regexp.MustCompile("<a href=\"/0/log/find\\.cgi\\?log=(.+?)\">DOWNLOAD</a>")
)

type crawler struct {
	userName    string
	storageType string
	storage     fs.FileStorage
	twitterCli  twitter.Client
}

func (c *crawler) getFiles(chFileName chan<- string) error {
	defer close(chFileName)

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
		chFileName <- value[1]
	}
	return nil
}

func (c *crawler) processFiles(chFileName <-chan string, chErrFile chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	netClient := &http.Client{
		Timeout: time.Second * 10,
	}

	for fileName := range chFileName {
		path := c.storage.GetPath(fmt.Sprintf("%s.mjlog", fileName))
		log.Printf("Start processing %s", path)
		exists, err := c.storage.Exists(path)
		if err != nil {
			log.Printf("Failed to check existence (%s): %v", fileName, err)
			chErrFile <- fileName
			continue
		}
		if exists {
			log.Printf("%s already exists", fileName)
		} else {
			response, err := netClient.Get(fmt.Sprintf("https://tenhou.net/0/log/find.cgi?log=%s", fileName))
			if err != nil {
				log.Printf("Failed to get mjlog (%s): %v", fileName, err)
				chErrFile <- fileName
				continue
			}
			var bs []byte
			func() {
				defer response.Body.Close()

				log.Println(fmt.Sprintf("Try saving %s into %s", path, c.storageType))
				bs, err = ioutil.ReadAll(response.Body)
				if err != nil {
					log.Printf("Failed to read response (%s): %v", fileName, err)
					chErrFile <- fileName
				}
			}()

			if err := c.storage.Save(path, bytes.NewBuffer(bs)); err != nil {
				log.Printf("Failed to save file (%s): %v", fileName, err)
				chErrFile <- fileName
				continue
			}

			mjlog, err := tenhou.Unmarshal(bytes.NewBuffer(bs))
			if err != nil {
				chErrFile <- fileName
				continue
			}
			msg := fmt.Sprintf("http://tenhou.net/0/?log=%s\n", fileName)
			msg += fmt.Sprintf("%s\n", mjlog.GameInfo.Name())
			for _, res := range mjlog.GetResult().Sort() {
				msg += fmt.Sprintf("%s %d (%s)\n", mjlog.Players[res.Player].Name, res.Ten, res.Point)
			}

			log.Printf("Tweet %s", msg)
			if err := c.twitterCli.Tweet(msg); err != nil {
				log.Printf("Failed to tweet (%s): %v", fileName, err)
				chErrFile <- fileName
				continue
			}
		}
	}
}

func (c *crawler) Crawl() error {
	log.Print("Start crawling..")
	chFileName := make(chan string)
	var wg sync.WaitGroup
	chErrFile := make(chan string)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go c.processFiles(chFileName, chErrFile, &wg)
	}

	err := c.getFiles(chFileName)
	if err != nil {
		return err
	}

	var errorFiles []string
	var wgErr sync.WaitGroup
	wgErr.Add(1)
	go func() {
		defer wgErr.Done()
		for errorFile := range chErrFile {
			errorFiles = append(errorFiles, errorFile)
		}
	}()

	wg.Wait()
	close(chErrFile)

	wgErr.Wait()
	if len(errorFiles) != 0 {
		return fmt.Errorf("failed to process the files (%s)", strings.Join(errorFiles, ", "))
	}
	return nil
}

func New(userName, storageType string, twitterCli twitter.Client) (*crawler, error) {
	c := new(crawler)
	c.userName = userName
	c.storageType = storageType
	c.twitterCli = twitterCli
	s, err := fs.New(storageType)
	if err != nil {
		return nil, err
	}
	c.storage = s
	return c, nil
}
