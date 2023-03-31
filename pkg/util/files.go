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

func IsUrlAlreadyPresent(url string) (bool, error) {
	path, err := GetUrlsFilePath()
	if err != nil {
		return false, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0664)
	if err != nil {
		return false, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return false, err
	}

	trimmedUrl := strings.TrimSpace(url)

	if stat.Size() != 0 {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if strings.TrimSpace(trimmedUrl) == trimmedUrl {
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

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0664)
	if err != nil {
		return err
	}
	defer f.Close()

	trimmedUrl := strings.TrimSpace(url)

	present, err := IsUrlAlreadyPresent(trimmedUrl)
	if err != nil {
		return err
	}
	if present {
		return fmt.Errorf("url already present!")
	}

	if _, err := f.WriteString(fmt.Sprintf("%s\n", trimmedUrl)); err != nil {
		return err
	}
	return nil
}
