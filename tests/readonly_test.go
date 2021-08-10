package tests

import (
	"testing"

	"github.com/erkkah/git-private/commands"
)

func TestReadonly(t *testing.T) {
	runAll(Suite{
		name: "readonly", tests: []NamedTest{
			{"adding keys fails", testReadonlyAddKeyFails},
			{"hiding fails", testReadonlyHidingFails},
			{"reveal succeeds", testReadonlyRevealSucceeds},
		},
	}, t)
}

func setupKeys(t *testing.T) {
	err := commands.Keys([]string{"add", "-id", "rw", "-pubfile", onePublicKey, "-keyfile", oneKey}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Keys([]string{"add", "-id", "ro", "-pubfile", anotherPublicKey, "-keyfile", oneKey, "-readonly"}, func() {})
	if err != nil {
		t.Fatal(err)
	}
}

func testReadonlyAddKeyFails(t *testing.T) {
	setupKeys(t)

	err := commands.Keys([]string{"add", "-id", "noway", "-pubfile", onePublicKey, "-keyfile", anotherKey}, func() {})
	if err == nil {
		t.Fatal("Adding public key with readonly key should fail!")
	}
}

func testReadonlyHidingFails(t *testing.T) {
	setupKeys(t)
	makeFile("secret", t)

	err := commands.Add([]string{"secret"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Hide([]string{"-keyfile", anotherKey}, func() {})
	if err == nil {
		t.Fatal("Hiding with readonly key should fail!")
	}
}

func testReadonlyRevealSucceeds(t *testing.T) {
	setupKeys(t)
	makeFile("secret", t)

	err := commands.Add([]string{"secret"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Hide([]string{"-keyfile", oneKey, "-clean"}, func() {})
	if err != nil {
		t.Fatal(err)
	}

	err = commands.Reveal([]string{"-keyfile", anotherKey}, func() {})
	if err != nil {
		t.Fatal(err)
	}
}
