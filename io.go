package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/spf13/afero"
)

var logsRoot = filepath.Join(os.Getenv("WORKSPACE"), "logs")

// appFs is an aero filesystem.  It is stored in a variable so that we can replace it
// with in-memory filesystems for unit tests.
var appFs = afero.NewOsFs()

type logger struct {
	cmdLogger    fileLogger
	testerLogger fileLogger
}

// Create a new logger that will log in the proper directory.
// Also initializes all necessary directories and files.
func newLogger() (logger, error) {
	logDir := filepath.Join(logsRoot, "log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return logger{}, err
	}

	return logger{
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
