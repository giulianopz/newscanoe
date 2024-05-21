package newscanoe

import (
	"os"

	"github.com/giulianopz/newscanoe/internal/util"
)

func RemoveCacheFile() error {

	cachePath, err := util.GetCacheFilePath()
	if err != nil {
		return err
	}
	return os.Remove(cachePath)
}
