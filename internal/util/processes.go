package util

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

/*
openWithBrowser opens the selected article with the default browser for desktop environment.
The environment variable DISPLAY is used to detect if the app is running on a headless machine.
see: https://wiki.debian.org/DefaultWebBrowser
*/
func OpenWithBrowser(url string) error {

	if url != "" {
		err := exec.Command("xdg-open", url).Run()
		if err != nil {
			switch e := err.(type) {
			case *exec.Error:
				return fmt.Errorf("failed executing: %v", err)
			case *exec.ExitError:
				return fmt.Errorf("command exit with code: %v", e.ExitCode())
			default:
				panic(err)
			}
		}
	}
	return nil
}

// headless detects if this is a headless machine by looking up the DISPLAY environment variable
func IsHeadless() bool {
	displayVar, set := os.LookupEnv("DISPLAY")
	log.Default().Printf("DISPLAY=%s", displayVar)
	return !set || displayVar == ""
}

func IsLynxPresent() bool {
	path, err := exec.LookPath("lynx")
	log.Default().Printf("lynx=%s", path)
	return err == nil && path != ""
}

/*
openWithLynx opens the selected article with lynx.
see: https://lynx.invisible-island.net/
*/
func OpenWithLynx(url string) error {

	cmd := exec.Command("lynx", url)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}
