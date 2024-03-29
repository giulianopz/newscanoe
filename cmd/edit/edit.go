package edit

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/giulianopz/newscanoe/internal/config"
	"github.com/giulianopz/newscanoe/internal/util"
	"gopkg.in/yaml.v3"
)

func EditConfigFile() error {

	configFilePath, err := util.GetConfigFilePath()
	if err != nil {
		return err
	}

	var editorName string = "vi"

	defaultEditorName, found := os.LookupEnv("EDITOR")
	if found {
		editorName = defaultEditorName
	}

	var retries int = 3

	for retries != 0 {

		retries--

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

		bs, err := os.ReadFile(configFilePath)
		if err != nil {
			return err
		}

		if err := yaml.Unmarshal(bs, &config.Config{}); err != nil {

			buf := bytes.Buffer{}
			buf.WriteString("# " + err.Error() + "\n")
			buf.Write(util.StripComments(bs))
			if err := os.WriteFile(configFilePath, buf.Bytes(), os.ModePerm); err != nil {
				return err
			}
		} else {

			if err := os.WriteFile(configFilePath, util.StripComments(bs), os.ModePerm); err != nil {
				return err
			}
			break
		}
	}

	if retries == 0 {
		return fmt.Errorf("max retries exceeded!")
	}

	return nil
}
