package tests

import (
	"testing"

	"github.com/erkkah/git-private/commands"
	"github.com/erkkah/git-private/utils"
)

func TestAdd(t *testing.T) {
	runAll(Suite{
		name: "arg", tests: []NamedTest{
			{"no args", testAddNoArgsFails},
			{"no file", testAddNonExistingFileFails},
			{"add single file", testAddSingleFileWorks},
		},
	}, t)
}

func testAddNoArgsFails(t *testing.T) {
	err := commands.Add([]string{}, func() {})
	if err == nil {
		t.Fatal(err)
	}
}

func testAddNonExistingFileFails(t *testing.T) {
	err := commands.Add([]string{"nosuchfile"}, func() {})
	if err == nil {
		t.Fatal(err)
	}
}

func testAddSingleFileWorks(t *testing.T) {
	makeFile("single", t)
	err := commands.Add([]string{"single"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	list, err := utils.LoadFileList()
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Files) != 1 || list.Files[0].Path != "single" {
		t.Fatal()
	}
}
