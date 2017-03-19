package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/afero"
)

// appFs is an aero filesystem.  It is stored in a variable so that we can replace it
// with in-memory filesystems for unit tests.
var appFs = afero.NewOsFs()

type logger struct {
	rootDir      string
	cmdLogger    fileLogger
	testerLogger fileLogger
	ip           string
}

// Creates a new fileLogger for the given test, in the appropriate
// "passed" or "failed" directory.
func (l logger) testLogger(passed bool, testName string) fileLogger {
	filename := fmt.Sprintf("%s.txt", testName)
	folder := filepath.Join(l.rootDir, "passed")
	if !passed {
		folder = filepath.Join(l.rootDir, "failed")
	}
	return fileLogger(filepath.Join(folder, filename))
}

// Retrieve the URL for the logger output.
func (l logger) url() string {
	return fmt.Sprintf("http://%s/%s", l.ip, filepath.Base(l.rootDir))
}

// Create a new logger that will log in the proper directory.
// Also initializes all necessary directories and files.
func newLogger(myIP string) (logger, error) {
	webDir := filepath.Join(webRoot, time.Now().Format("02-01-2006_15h04m05s"))
	passedDir := filepath.Join(webDir, "passed")
	failedDir := filepath.Join(webDir, "failed")
	logDir := filepath.Join(webDir, "log")
	buildinfoPath := filepath.Join(webDir, "buildinfo")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return logger{}, err
	}
	if err := os.MkdirAll(passedDir, 0755); err != nil {
		return logger{}, err
	}
	if err := os.MkdirAll(failedDir, 0755); err != nil {
		return logger{}, err
	}
	if err := exec.Command("cp", "/buildinfo", buildinfoPath).Run(); err != nil {
		logrus.WithError(err).Error("Failed to copy build info.")
	}

	latestSymlink := filepath.Join(webRoot, "latest")
	os.Remove(latestSymlink)
	if err := os.Symlink(webDir, latestSymlink); err != nil {
		return logger{}, err
	}

	return logger{
		ip:           myIP,
		rootDir:      webDir,
		testerLogger: fileLogger(filepath.Join(logDir, "quilt-tester.log")),
		cmdLogger:    fileLogger(filepath.Join(logDir, "container.log")),
	}, nil
}

type fileLogger string

func (l fileLogger) infoln(msg string) {
	timestamp := time.Now().Format("[15:04:05] ")
	toWrite := "\n" + timestamp + "=== " + msg + " ===\n"
	if err := writeTo(string(l), toWrite); err != nil {
		logrus.WithError(err).Errorf("Failed to write %s to %s.", msg, string(l))
	}
}

func (l fileLogger) errorln(msg string) {
	toWrite := "\n=== Error Text ===\n" + msg + "\n"
	if err := writeTo(string(l), toWrite); err != nil {
		logrus.WithError(err).Errorf("Failed to write %s to %s.", msg, string(l))
	}
}

func (l fileLogger) println(msg string) {
	if err := writeTo(string(l), msg+"\n"); err != nil {
		logrus.WithError(err).Errorf("Failed to write %s to %s.", msg, string(l))
	}
}

func writeTo(file string, message string) error {
	a := afero.Afero{
		Fs: appFs,
	}

	f, err := a.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.WithError(err).Errorf("Couldn't open %s for writing", file)
		return err
	}

	defer f.Close()
	_, err = f.WriteString(message)
	return err
}

func overwrite(file string, message string) error {
	a := afero.Afero{
		Fs: appFs,
	}
	return a.WriteFile(file, []byte(message), 0666)
}

func fileContents(file string) (string, error) {
	a := afero.Afero{
		Fs: appFs,
	}
	contents, err := a.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

type message struct {
	Title string `json:"title"`
	Short bool   `json:"short"`
	Value string `json:"value"`
}

type slackPost struct {
	Channel   string    `json:"channel"`
	Color     string    `json:"color"`
	Fields    []message `json:"fields"`
	Pretext   string    `json:"pretext"`
	Username  string    `json:"username"`
	Iconemoji string    `json:"icon_emoji"`
}

// Post to slack.
func slack(hookurl string, p slackPost) error {
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	resp, err := http.Post(hookurl, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(t))
	}

	return nil
}

// Update the given spec to have the given namespace.
func updateNamespace(specfile string, namespace string) error {
	specContents, err := fileContents(specfile)
	if err != nil {
		return err
	}

	// Set the namespace of the global deployment to be `namespace`.
	updatedSpec := specContents +
		fmt.Sprintf("; deployment.namespace = %q;", namespace)

	return overwrite(specfile, updatedSpec)
}
