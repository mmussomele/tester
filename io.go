package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"
)

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
	cmdLoggerPath := filepath.Join(os.Getenv("WORKSPACE"), "commandOutputs.log")
	cmdLoggerFile, err := os.Create(cmdLoggerPath)
	if err != nil {
		return logger{}, err
	}

	return logger{
		testerLogger: fileLogger{os.Stdout},
		cmdLogger:    fileLogger{cmdLoggerFile},
	}, nil
}

type fileLogger struct {
	out io.Writer
}

func (l fileLogger) infoln(msg string) {
	timestamp := time.Now().Format("[15:04:05] ")
	l.println("\n" + timestamp + "=== " + msg + " ===")
}

func (l fileLogger) errorln(msg string) {
	l.println("\n=== Error Text ===\n" + msg + "\n")
}

func (l fileLogger) println(msg string) {
	fmt.Fprintln(l.out, msg)
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
