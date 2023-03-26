package util

import (
	"os"
	"path/filepath"
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
	return filepath.Join(dirName, urlsFileName), nil
}

func GetCacheFilePath() (string, error) {
	dirName, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirName, cacheFileName), nil
}
