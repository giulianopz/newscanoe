package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/giulianopz/newscanoe/cmd/newscanoe"
)

var (
	debugFlag       bool
	editFlag        bool
	removeCacheFlag bool
)

const usage = `Usage:
    newscanoe [OPTION]...

Options:
	-d, --debug		Enable debug mode.
	-e, --edit		Edit config file with default text editor (according to $EDITOR).
	-c, --clean		Remove cache file.
`

func main() {

	flag.BoolVar(&debugFlag, "d", false, "enable debug mode")
	flag.BoolVar(&debugFlag, "debug", false, "enable debug mode")
	flag.BoolVar(&editFlag, "e", false, "edit config file with default text editor (according to $EDITOR)")
	flag.BoolVar(&editFlag, "edit", false, "edit config file with default text editor (according to $EDITOR)")
	flag.BoolVar(&removeCacheFlag, "c", false, "remove cache file")
	flag.BoolVar(&removeCacheFlag, "clean", false, "remove cache file")
	flag.Usage = func() { fmt.Print(usage) }
	flag.Parse()

	if !debugFlag {
		log.SetOutput(io.Discard)
	}

	var err error

	if editFlag {
		err = newscanoe.EditConfigFile()
	} else if removeCacheFlag {
		err = newscanoe.RemoveCacheFile()
	} else {
		newscanoe.Run(debugFlag)
	}

	if err != nil {
		log.Default().Println(err)
		os.Exit(1)
	}
}
