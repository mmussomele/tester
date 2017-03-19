package main

import (
	"fmt"
	"os/exec"
	"sort"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCmdExec(t *testing.T) {
	appFs = afero.NewMemMapFs()

	outputPath := "output.log"
	log = logger{
		cmdLogger: fileLogger(outputPath),
	}

	expStdout := "standard out"
	expStderr := "standard error"
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("echo %s ; echo %s 1>&2", expStdout, expStderr))
	stdout, stderr, err := execCmd(cmd, "PREFIX")

	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if stdout != expStdout {
		t.Errorf("Stdout didn't match: expected %s, got %s", expStdout, stdout)
	}
	if stderr != expStderr {
		t.Errorf("Stderr didn't match: expected %s, got %s", expStderr, stderr)
	}
}

func TestUpdateNamespace(t *testing.T) {
	appFs = afero.NewMemMapFs()

	specPath := "/test.spec"
	err := overwrite(specPath, `require("spark");
var deployment = createDeployment({namespace: "replace"});
deployment.deploy(new Machine({}));`)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	updateNamespace(specPath, "test-namespace")

	res, err := fileContents(specPath)
	exp := `require("spark");
var deployment = createDeployment({namespace: "replace"});
deployment.deploy(new Machine({}));; deployment.namespace = "test-namespace";`
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if res != exp {
		t.Errorf("Namespace didn't properly update, expected %s, got %s",
			exp, res)
	}
}

func checkSort(t *testing.T, toSort, exp []string) {
	sort.Sort(byPriorityPrefix(toSort))
	assert.Equal(t, exp, toSort)
}

func TestPriorityPrefix(t *testing.T) {
	// Order based on priority.
	checkSort(t, []string{"100-foo", "50-bar"}, []string{"50-bar", "100-foo"})
	checkSort(t, []string{"50-bar", "100-foo"}, []string{"50-bar", "100-foo"})

	// Order based on name.
	checkSort(t, []string{"50-foo", "50-bar"}, []string{"50-bar", "50-foo"})

	// Default priority.
	checkSort(t, []string{"foo", "20-bar"}, []string{"20-bar", "foo"})
	checkSort(t, []string{"foo", "100-bar"}, []string{"foo", "100-bar"})
}
