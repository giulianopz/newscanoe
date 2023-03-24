package util

import (
	"os"
	"path/filepath"

	"github.com/giulianopz/newscanoe/pkg/app"
)

const (
	xdgConfigDirEnvVar   = "XDG_CONFIG_HOME"
	xdgCacheDirEnvVar    = "XDG_CACHE_HOME"
	defaultConfigDirName = ".config"
	defaultCacheDirName  = ".cache"
	urlsFileName         = "urls"
	cacheFileName        = "cache.gob"
)

func GetUrlsFilePath() (string, error) {
	dirName, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirName, urlsFileName), nil
}

func GetCacheFilePath() (string, error) {
	dirName, err := getConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dirName, cacheFileName), nil
}

func getConfigDir() (string, error) {
	return getDir(xdgConfigDirEnvVar, defaultConfigDirName)
}

func getCacheDir() (string, error) {
	return getDir(xdgCacheDirEnvVar, defaultCacheDirName)
}

func getDir(xdgEnvVar, defaultName string) (string, error) {

	dir := os.Getenv(xdgEnvVar)
	if dir == "" || dir[0:1] != "/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, defaultName, app.Name), nil
	}
	return filepath.Join(dir, app.Name), nil
}
