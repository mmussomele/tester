package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"

	"github.com/quilt/quilt/api"
	"github.com/quilt/quilt/api/client/getter"
	apiUtil "github.com/quilt/quilt/api/util"
	"github.com/quilt/quilt/stitch"
	"github.com/quilt/quilt/util"
)

var (
	quiltPath          = filepath.Join(os.Getenv("WORKSPACE"), ".quilt")
	testerImport       = "github.com/quilt/tester"
	infrastructureSpec = filepath.Join(quiltPath, testerImport,
		"config/infrastructure-runner.js")
)

// The global logger for this CI run.
var log logger

func main() {
	namespace := os.Getenv("TESTING_NAMESPACE")
	if namespace == "" {
		logrus.Error("Please set TESTING_NAMESPACE.")
		os.Exit(1)
	}

	var err error
	if log, err = newLogger(); err != nil {
		logrus.WithError(err).Error("Failed to create logger.")
		os.Exit(1)
	}

	tester, err := newTester(namespace)
	if err != nil {
		logrus.WithError(err).Error("Failed to create tester instance.")
		os.Exit(1)
	}

	if err := tester.run(); err != nil {
		logrus.WithError(err).Error("Test execution failed.")
		os.Exit(1)
	}
}

type tester struct {
	preserveFailed bool
	junitOut       string

	testSuites  []*testSuite
	initialized bool
	namespace   string
}

func newTester(namespace string) (tester, error) {
	t := tester{
		namespace: namespace,
	}

	testRoot := flag.String("testRoot", "",
		"the root directory containing the integration tests")
	flag.BoolVar(&t.preserveFailed, "preserve-failed", false,
		"don't destroy machines on failed tests")
	flag.StringVar(&t.junitOut, "junitOut", "",
		"location to write junit report")
	flag.Parse()

	if *testRoot == "" {
		return tester{}, errors.New("testRoot is required")
	}

	err := t.generateTestSuites(*testRoot)
	if err != nil {
		return tester{}, err
	}

	return t, nil
}

func (t *tester) generateTestSuites(testRoot string) error {
	l := log.testerLogger

	// First, we need to ls the testRoot, and find all of the folders. Then we can
	// generate a testSuite for each folder.
	testSuiteFolders, err := filepath.Glob(filepath.Join(testRoot, "*"))
	if err != nil {
		l.infoln("Could not access test suite folders")
		l.errorln(err.Error())
		return err
	}

	sort.Sort(byPriorityPrefix(testSuiteFolders))
	for _, testSuiteFolder := range testSuiteFolders {
		files, err := ioutil.ReadDir(testSuiteFolder)
		if err != nil {
			l.infoln(fmt.Sprintf(
				"Error reading test suite %s", testSuiteFolder))
			l.errorln(err.Error())
			return err
		}

		var spec, test string
		for _, file := range files {
			path := filepath.Join(testSuiteFolder, file.Name())
			switch {
			case strings.HasSuffix(file.Name(), ".js"):
				spec = path
				if err := updateNamespace(spec, t.namespace); err != nil {
					l.infoln(fmt.Sprintf(
						"Error updating namespace for %s.", spec))
					l.errorln(err.Error())
					return err
				}
			// If the file is executable by everyone, and is not a directory.
			case (file.Mode()&1 != 0) && !file.IsDir():
				test = path
			}
		}
		newSuite := testSuite{
			name: filepath.Base(testSuiteFolder),
			spec: spec,
			test: test,
		}
		t.testSuites = append(t.testSuites, &newSuite)
	}

	return nil
}

func (t tester) run() error {
	defer func() {
		if t.junitOut != "" {
			writeJUnitReport(t.testSuites, t.junitOut)
		}

		failed := false
		for _, suite := range t.testSuites {
			if !suite.passed {
				failed = true
				break
			}
		}

		if failed && t.preserveFailed {
			return
		}

		cleanupMachines(t.namespace)
	}()

	if err := t.setup(); err != nil {
		log.testerLogger.errorln("Unable to setup the tests, bailing.")
		// All suites failed if we didn't run them.
		for _, suite := range t.testSuites {
			suite.passed = false
		}
		return err
	}

	return t.runTestSuites()
}

func (t *tester) setup() error {
	l := log.testerLogger

	l.infoln("Starting the Quilt daemon.")
	go runQuiltDaemon()

	// Get our specs
	os.Setenv(stitch.QuiltPathKey, quiltPath)
	l.infoln(fmt.Sprintf("Downloading %s into %s", testerImport, quiltPath))
	_, _, err := downloadSpecs(testerImport)
	if err != nil {
		l.infoln(fmt.Sprintf("Could not download %s", testerImport))
		l.errorln(err.Error())
		return err
	}

	// Do a preliminary quilt stop.
	l.infoln(fmt.Sprintf("Preliminary `quilt stop %s`", t.namespace))
	_, _, err = stop(t.namespace)
	if err != nil {
		l.infoln(fmt.Sprintf("Error stopping: %s", err.Error()))
		return err
	}

	// Setup infrastructure.
	l.infoln("Booting the machines the test suites will run on, and waiting " +
		"for them to connect back.")
	l.infoln("Begin " + infrastructureSpec)
	if err := updateNamespace(infrastructureSpec, t.namespace); err != nil {
		l.infoln(fmt.Sprintf("Error updating namespace for %s.",
			infrastructureSpec))
		l.errorln(err.Error())
		return err
	}
	contents, _ := fileContents(infrastructureSpec)
	l.println(contents)
	l.infoln("End " + infrastructureSpec)

	_, _, err = runSpecUntilConnected(infrastructureSpec)
	if err != nil {
		l.infoln("Failed to setup infrastructure")
		l.errorln(err.Error())
		return err
	}

	l.infoln("Booted Quilt")
	l.infoln("Machines")
	machines, _ := queryMachines()
	l.println(fmt.Sprintf("%v", machines))

	return nil
}

func (t tester) runTestSuites() error {
	var err error
	for _, suite := range t.testSuites {
		if e := suite.run(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

type testSuite struct {
	name string
	spec string
	test string

	output      string
	passed      bool
	timeElapsed time.Duration
}

func (ts *testSuite) run() error {
	testStart := time.Now()
	defer func() {
		ts.timeElapsed = time.Since(testStart)
	}()

	l := log.testerLogger

	l.infoln(fmt.Sprintf("Test Suite: %s", ts.name))
	l.infoln("Start " + ts.name + ".js")
	contents, _ := fileContents(ts.spec)
	l.println(contents)
	l.infoln("End " + ts.name + ".js")
	defer l.infoln(fmt.Sprintf("Finished Test Suite: %s", ts.name))

	runSpec(ts.spec)

	l.infoln("Waiting for containers to start up")
	if err := waitForContainers(ts.spec); err != nil {
		l.println(".. Containers never started: " + err.Error())
		ts.passed = false
		return err
	}

	// Wait a little bit longer for any container bootstrapping after boot.
	time.Sleep(30 * time.Second)

	var err error
	if ts.test != "" {
		l.infoln("Starting Test")
		l.println(".. " + filepath.Base(ts.test))

		ts.output, err = runTest(ts.test)
		if err == nil {
			l.println(".... Passed")
			ts.passed = true
		} else {
			l.println(".... Failed")
			ts.passed = false
		}
	}

	return err
}

func waitForContainers(specPath string) error {
	stc, err := stitch.FromFile(specPath, stitch.NewImportGetter(quiltPath))
	if err != nil {
		return err
	}

	localClient, err := getter.New().Client(api.DefaultSocket)
	if err != nil {
		return err
	}

	return util.WaitFor(func() bool {
		for _, exp := range stc.Containers {
			containerClient, err := getter.New().ContainerClient(localClient,
				exp.ID)
			if err != nil {
				return false
			}

			actual, err := apiUtil.GetContainer(containerClient, exp.ID)
			if err != nil || actual.Created.IsZero() {
				return false
			}
		}
		return true
	}, 15*time.Second, 10*time.Minute)
}

func runTest(testPath string) (string, error) {
	output, err := exec.Command(testPath).CombinedOutput()
	if err != nil || !strings.Contains(string(output), "PASSED") {
		_, testName := filepath.Split(testPath)
		err = fmt.Errorf("test failed: %s", testName)
	}
	return string(output), err
}
