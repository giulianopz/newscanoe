package util

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/giulianopz/newscanoe/internal/app"
)

const (
	configFileName = "config.yaml"
	cacheFileName  = "feeds.gob"
)

func GetConfigFilePath() (string, error) {
	configDirName, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	log.Default().Printf("found config dir: %q\n", configDirName)

	appConfigDirName := filepath.Join(configDirName, app.Name)
	if !Exists(appConfigDirName) {
		if err := os.Mkdir(appConfigDirName, 0777); err != nil {
			return "", err
		}
	}

	urlsFilePath := filepath.Join(appConfigDirName, configFileName)

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
