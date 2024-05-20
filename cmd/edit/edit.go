package edit

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/giulianopz/newscanoe/internal/util"
)

var linePattern = regexp.MustCompile(`http[^"]+((\s#"[^"]+")+)`)

const errMsg = "# the following line does not respect the pattern"

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

	for retries := 3; retries > 0; retries-- {
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

		f, err := os.OpenFile(configFilePath, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			return err
		}
		defer f.Close()

		bs, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		fileIsValid := true
		buf := bytes.Buffer{}

		s := bufio.NewScanner(bytes.NewReader(bs))
		for s.Scan() {
			line := s.Text()
			if !linePattern.MatchString(line) && !strings.HasPrefix(line, "#") {
				fileIsValid = false
				if _, err := buf.WriteString(fmt.Sprintf("%s: %s\n", errMsg, line)); err != nil {
					return err
				}
				break
			}
		}
		if err := s.Err(); err != nil {
			return err
		}

		if _, err := buf.Write(util.RemoveLines(bs, func(s string) bool {
			return strings.HasPrefix(s, errMsg)
		})); err != nil {
			return err
		}
		if err := os.WriteFile(configFilePath, buf.Bytes(), os.ModePerm); err != nil {
			return err
		}

		if fileIsValid {
			break
		}

		if retries == 0 {
			return fmt.Errorf("max retries exceeded")
		}
	}

	return nil
}
