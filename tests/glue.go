package tests

import (
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path"
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

func copyFile(src string, dst string, t *testing.T) {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile(dst, data, 0600)
	if err != nil {
		t.Fatal(err)
	}
}

const oneKey = "one.key"
const onePublicKey = "one.pub"
const anotherKey = "another.key"
const anotherPublicKey = "another.pub"

var cwd string

func init() {
	cwd, _ = os.Getwd()
}

func setupAndInit(t *testing.T) {
	var err error

	testDir := t.TempDir()

	err = os.Chdir(cwd)
	if err != nil {
		t.Fatalf("Failed to move to test directory: %v", err)
	}

	for _, keyFile := range []string{oneKey, onePublicKey, anotherKey, anotherPublicKey} {
		copyFile(path.Join("testkeys", keyFile), path.Join(testDir, keyFile), t)
	}

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Failed to move to tmp directory: %v", err)
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
