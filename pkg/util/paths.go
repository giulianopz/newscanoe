package util

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

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
	return filepath.Join(appConfigDirName, urlsFileName), nil
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
