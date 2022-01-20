package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
)

const (
	// applicationId — A string used as the unique application ID for the app on Google Play Store [...] -> https://developer.android.com/studio/build/configure-app-module
	applicationIDRegexPattern = `^applicationId(?:=|\s)+(.*?)(?:\s|\/\/|$)`
)

type config struct {
	BuildGradlePth   string `env:"build_gradle_path,file"`
	NewApplicationID string `env:"new_application_id"`
}

type updateFn func(line string, lineNum int, matches []string) string

func findAndUpdate(reader io.Reader, update map[*regexp.Regexp]updateFn) (string, error) {
	scanner := bufio.NewScanner(reader)
	var updatedLines []string

	for lineNum := 0; scanner.Scan(); lineNum++ {
		line := scanner.Text()

		updated := false
		for re, fn := range update {
			if match := re.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 2 {
				if updatedLine := fn(line, lineNum, match); updatedLine != "" {
					updatedLines = append(updatedLines, updatedLine)
					updated = true
					break
				}
			}
		}
		if !updated {
			updatedLines = append(updatedLines, line)
		}
	}

	return strings.Join(updatedLines, "\n"), scanner.Err()
}

func exportOutputs(outputs map[string]string) error {
	for envKey, envValue := range outputs {
		cmd := command.New("envman", "add", "--key", envKey, "--value", envValue)
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

// BuildGradleApplicationIDUpdater updates applicationId in the given build.gradle file.
type BuildGradleApplicationIDUpdater struct {
	buildGradleReader io.Reader
}

// NewBuildGradleApplicationIDUpdater constructs a new NewBuildGradleApplicationIDUpdater.
func NewBuildGradleApplicationIDUpdater(buildGradleReader io.Reader) BuildGradleApplicationIDUpdater {
	return BuildGradleApplicationIDUpdater{buildGradleReader: buildGradleReader}
}

// UpdateResult stors the result of the applicationId update.
type UpdateResult struct {
	NewContent           string
	FinalApplicationID   string
	UpdatedApplicationID int
}

// UpdateApplicationID executes the applicationId update.
func (u BuildGradleApplicationIDUpdater) UpdateApplicationID(newApplicationID string) (UpdateResult, error) {
	res := UpdateResult{}
	var err error

	res.NewContent, err = findAndUpdate(u.buildGradleReader, map[*regexp.Regexp]updateFn{
		regexp.MustCompile(applicationIDRegexPattern): func(line string, lineNum int, match []string) string {
			oldApplicationID := match[1]
			res.FinalApplicationID = oldApplicationID
			updatedLine := ""

			if newApplicationID != "" {
				quotedNewApplicationID := newApplicationID
				if !(strings.HasPrefix(quotedNewApplicationID, `"`) && strings.HasSuffix(quotedNewApplicationID, `"`)) {
					quotedNewApplicationID = strings.TrimPrefix(quotedNewApplicationID, `"`)
					quotedNewApplicationID = strings.TrimSuffix(quotedNewApplicationID, `"`)
					quotedNewApplicationID = `"` + quotedNewApplicationID + `"`
					log.Warnf(`Leading and/or trailing " character missing from new_application_id, adding quotation char: %s -> %s`, newApplicationID, quotedNewApplicationID)
				}

				res.FinalApplicationID = quotedNewApplicationID
				updatedLine = strings.Replace(line, oldApplicationID, res.FinalApplicationID, -1)
				res.UpdatedApplicationID++
				log.Printf("updating line (%d): %s -> %s", lineNum, line, updatedLine)
			}

			return updatedLine
		},
	})
	if err != nil {
		return UpdateResult{}, err
	}
	return res, nil
}

func main() {
	var cfg config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(cfg)
	fmt.Println()

	if cfg.NewApplicationID == "" {
		failf("NewApplicationID is required.")
	}

	//
	// find applicationId with regexp
	fmt.Println()
	log.Infof("Updating applicationId in: %s", cfg.BuildGradlePth)

	f, err := os.Open(cfg.BuildGradlePth)
	if err != nil {
		failf("Failed to read build.gradle file, error: %s", err)
	}

	applicationIDUpdater := NewBuildGradleApplicationIDUpdater(f)
	res, err := applicationIDUpdater.UpdateApplicationID(cfg.NewApplicationID)
	if err != nil {
		failf("Failed to update applicationId: %s", err)
	}

	//
	// export outputs
	if err := exportOutputs(map[string]string{
		"ANDROID_APPLICATION_ID": removeQuotationMarks(res.FinalApplicationID),
	}); err != nil {
		failf("Failed to export outputs, error: %s", err)
	}

	if err := fileutil.WriteStringToFile(cfg.BuildGradlePth, res.NewContent); err != nil {
		failf("Failed to write build.gradle file, error: %s", err)
	}

	fmt.Println()
	log.Donef("%d applicationId updated", res.UpdatedApplicationID)
}

func removeQuotationMarks(value string) string {
	return strings.Trim(value, `"'`)
}
