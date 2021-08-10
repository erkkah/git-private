package tests

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"testing"

	"github.com/erkkah/git-private/commands"
)

type NamedTest struct {
	name string
	test func(*testing.T)
}

type Suite struct {
	name  string
	tests []NamedTest
}

func setupAndInit(t *testing.T) {
	testDir := t.TempDir()
	err := os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to move to test directory: %v", err)
	}

	git := exec.Command("git", "init")
	err = git.Run()
	if err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	err = commands.Init([]string{}, func() {})
	if err != nil {
		t.Fatalf("Failed to init: %v", err)
	}
}

func makeFile(name string, t *testing.T) {
	data := make([]byte, 5150)
	rand.Read(data)
	if ioutil.WriteFile(name, data, 0660) != nil {
		t.Fatalf("Failed to create test file")
	}
}

func runAll(suite Suite, t *testing.T) {
	for _, test := range suite.tests {
		setupAndInit(t)
		if !t.Run(suite.name+" - "+test.name, test.test) {
			break
		}
	}
}
