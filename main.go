package main

import (
	"flag"
	"io"
	"log"

	"github.com/giulianopz/newscanoe/cmd/edit"
	"github.com/giulianopz/newscanoe/cmd/newscanoe"
)

var (
	debugFlag bool
	editFlag  bool
)

func main() {
	flag.BoolVar(&debugFlag, "d", false, "enable debug mode")
	flag.BoolVar(&editFlag, "e", false, "edit config file with default text editor (according to $EDITOR)")
	flag.Parse()

	if !debugFlag {
		log.SetOutput(io.Discard)
	}

	if editFlag {
		if err := edit.EditConfigFile(); err != nil {
			log.Default().Println(err)
		}
		return
	}

	newscanoe.Run(debugFlag)
}
