package edit

import (
	"os"
	"os/exec"

	"github.com/giulianopz/newscanoe/internal/util"
)

func ConfigFile() error {

	configFilePath, err := util.GetConfigFilePath()
	if err != nil {
		return err
	}

	var editorName string = "vi"

	defaultEditorName, found := os.LookupEnv("EDITOR")
	if found {
		editorName = defaultEditorName
	}

	editor := exec.Command(editorName, configFilePath)
	editor.Stdout = os.Stdout
	editor.Stderr = os.Stderr
	editor.Stdin = os.Stdin

	if err := editor.Start(); err != nil {
		return err
	}

	if err := editor.Wait(); err != nil {
		return err
	}
	return nil
}
