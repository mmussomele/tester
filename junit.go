package main

import (
	"encoding/xml"
	"os"

	"github.com/Sirupsen/logrus"
)

type JUnitReport struct {
	XMLName     xml.Name `xml:"testsuite"`
	NumTests    int      `xml:"tests,attr"`
	TestResults []TestCase
}

type TestCase struct {
	XMLName   xml.Name  `xml:"testcase"`
	Name      string    `xml:"name,attr"`
	ClassName string    `xml:"classname,attr"`
	Failure   *struct{} `xml:"failure,omitempty"`
	Output    string    `xml:"system-out"`
}

func writeJUnitReport(tests []*testSuite, filename string) {
	report := JUnitReport{NumTests: len(tests)}
	for _, t := range tests {
		// Ignore test suites that are solely for setup, and do not test anything.
		if t.test == "" {
			continue
		}

		junitResult := TestCase{Name: t.name, ClassName: "tests", Output: t.output}
		if !t.passed {
			junitResult.Failure = &struct{}{}
		}
		report.TestResults = append(report.TestResults, junitResult)
	}

	f, err := os.Create(filename)
	if err != nil {
		logrus.WithError(err).Errorf(
			"Failed to create output file %s for test results", filename)
		return
	}

	enc := xml.NewEncoder(f)
	enc.Indent("  ", "    ")
	if err := enc.Encode(&report); err != nil {
		logrus.WithError(err).Error("Failed to marshal report")
		return
	}
}
