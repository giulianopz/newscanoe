package util

import (
	"os"
	"path/filepath"

	"github.com/giulianopz/newscanoe/pkg/app"
)

const (
	urlsFileName  = "urls"
	cacheFileName = "cache.gob"
)

func GetUrlsFilePath() (string, error) {
	dirName, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirName, app.Name, urlsFileName), nil
}

func GetCacheFilePath() (string, error) {
	dirName, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirName, app.Name, cacheFileName), nil
}
