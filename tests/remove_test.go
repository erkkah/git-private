package tests

import (
	"testing"

	"github.com/erkkah/git-private/commands"
	"github.com/erkkah/git-private/utils"
)

func TestRemove(t *testing.T) {
	runAll(Suite{
		name: "remove", tests: []NamedTest{
			{"no args", testRemoveNoArgsFails},
			{"no file", testRemoveNonExistingFileSucceeds},
			{"non-revealed file", testRemoveNonRevealedFileSucceeds},
		},
	}, t)
}

func testRemoveNoArgsFails(t *testing.T) {
	err := commands.Remove([]string{}, func() {})
	if err == nil {
		t.Fatal(err)
	}
}

func testRemoveNonExistingFileSucceeds(t *testing.T) {
	err := commands.Remove([]string{"nosuchfile"}, func() {})
	if err != nil {
		t.Fatal(err)
	}
}

func testRemoveNonRevealedFileSucceeds(t *testing.T) {
	makeFile("mysecrets", t)

	err := commands.Keys([]string{"add", "-id", "hubba", "-pubfile", onePublicKey, "-keyfile", oneKey}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Add([]string{"mysecrets"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Hide([]string{"-keyfile", oneKey, "-clean"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Remove([]string{"mysecrets"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	exists, err := utils.Exists("mysecrets.private")
	if err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("private file left behind")
	}
}
