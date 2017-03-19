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
}

func writeJUnitReport(tests []*testSuite, filename string) {
	report := JUnitReport{NumTests: len(tests)}
	for _, t := range tests {
		junitResult := TestCase{Name: t.name, ClassName: "tests"}
		if t.failed != 0 {
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
