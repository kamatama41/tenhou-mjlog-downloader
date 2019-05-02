package file_storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"github.com/kamatama41/tenhou-mjlog-downloader/env"
)

type FileStorage interface {
	GetPath(fileName string) string
	Exists(path string) (bool, error)
	Save(path string, file io.Reader) error
}

type dropboxStorage struct {
	token string
}

func (d *dropboxStorage) GetPath(fileName string) string {
	return fmt.Sprintf("/%s", fileName)
}

func (d *dropboxStorage) Exists(path string) (bool, error) {
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

func (d *dropboxStorage) Save(path string, file io.Reader) error {
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

func (l *localStorage) GetPath(fileName string) string {
	return fmt.Sprintf("%s/%s", l.basePath, fileName)
}

func (l *localStorage) Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	return !os.IsNotExist(err), nil
}

func (l *localStorage) Save(path string, file io.Reader) error {
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

func New(storageType string) (FileStorage, error) {
	switch storageType {
	case "dropbox":
		t, err := env.GetOrError("DROPBOX_API_TOKEN")
		if err != nil {
			return nil, err
		}
		return &dropboxStorage{
			token: t,
		}, nil
	default:
		return &localStorage{
			basePath: "tmp",
		}, nil
	}
}
