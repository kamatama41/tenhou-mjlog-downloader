package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
)

type fileStorage interface {
	getPath(fileName string) string
	exists(path string) (bool, error)
	save(path string, file io.Reader) error
}

func newStorage(storageType string) fileStorage {
	switch storageType {
	case "dropbox":
		return &dropboxStorage{
			token: getEnv("DROPBOX_API_TOKEN"),
		}
	default:
		return &localStorage{
			basePath: "tmp",
		}
	}
}

type dropboxStorage struct {
	token string
}

func (d *dropboxStorage) getPath(fileName string) string {
	return fmt.Sprintf("/%s", fileName)
}

func (d *dropboxStorage) exists(path string) (bool, error) {
	f := files.New(d.config())
	_, err := f.GetMetadata(&files.GetMetadataArg{
		Path: path,
	})
	if err != nil {
		metadataErr, ok := err.(files.GetMetadataAPIError)
		if ok && metadataErr.EndpointError.Path.Tag == "not_found" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (d *dropboxStorage) save(path string, file io.Reader) error {
	f := files.New(d.config())
	commitInfo := &files.CommitInfo{
		Path: path,
		Mode: &files.WriteMode{
			Tagged: dropbox.Tagged{Tag: files.WriteModeAdd},
		},
	}
	_, err := f.Upload(commitInfo, file)
	return err
}

func (d *dropboxStorage) config() dropbox.Config {
	return dropbox.Config{
		Token: d.token,
	}
}

type localStorage struct {
	basePath string
}

func (l *localStorage) getPath(fileName string) string {
	return fmt.Sprintf("%s/%s", l.basePath, fileName)
}

func (l *localStorage) exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return !os.IsNotExist(err), nil
}

func (l *localStorage) save(path string, file io.Reader) error {
	if _, err := os.Stat(l.basePath); os.IsNotExist(err) {
		err := os.Mkdir(l.basePath, 0755)
		if err != nil {
			return err
		}
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, bytes, 0644)
}
