package util

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/giulianopz/newscanoe/pkg/app"
)

const (
	urlsFileName  = "urls"
	cacheFileName = "feeds.gob"
)

func GetUrlsFilePath() (string, error) {
	configDirName, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	appConfigDirName := filepath.Join(configDirName, app.Name)
	if !Exists(appConfigDirName) {
		if err := os.Mkdir(appConfigDirName, 0777); err != nil {
			return "", err
		}
	}

	urlsFilePath := filepath.Join(appConfigDirName, urlsFileName)

	if !Exists(urlsFilePath) {
		if _, err := os.Create(urlsFilePath); err != nil {
			return "", err
		}
	}

	return urlsFilePath, nil
}

func GetCacheFilePath() (string, error) {
	cacheDirName, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	appCacheDirName := filepath.Join(cacheDirName, app.Name)
	if !Exists(appCacheDirName) {
		if err := os.Mkdir(appCacheDirName, 0777); err != nil {
			return "", err
		}
	}
	return filepath.Join(appCacheDirName, cacheFileName), nil
}

func Exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	} else if errors.Is(err, fs.ErrNotExist) {
		return false
	} else {
		log.Panicln(err)
		return false
	}
}

func IsEmpty(f *os.File) (bool, error) {
	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	return info.Size() == 0, nil
}

func isAlreadyPresent(url string, f *os.File) (bool, error) {
	stat, err := f.Stat()
	if err != nil {
		return false, err
	}

	if stat.Size() != 0 {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {

			if err := scanner.Err(); err != nil {
				log.Default().Printf("scan failed due to: %v\n", err)
				return false, err
			}

			log.Default().Printf("%s vs %s", strings.TrimSpace(url), strings.TrimSpace(scanner.Text()))
			if strings.TrimSpace(url) == strings.TrimSpace(scanner.Text()) {
				return true, nil
			}
		}
	}
	return false, nil
}

func AppendUrl(url string) error {

	path, err := GetUrlsFilePath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_RDWR, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	trimmedUrl := strings.TrimSpace(url)

	alreadyPresent, err := isAlreadyPresent(trimmedUrl, f)
	if err != nil {
		return err
	} else if alreadyPresent {
		return newUrlAlreadyPresentErr(url)
	}

	if _, err := f.WriteString(fmt.Sprintf("%s\n", trimmedUrl)); err != nil {
		return err
	}
	return nil
}

type UrlAlreadyPresentErr struct {
	url string
}

func (e *UrlAlreadyPresentErr) Error() string {
	return fmt.Sprintf("url is already present: %s", e.url)
}

func newUrlAlreadyPresentErr(url string) *UrlAlreadyPresentErr {
	return &UrlAlreadyPresentErr{url: url}
}
