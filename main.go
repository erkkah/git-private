package main

import (
	"fmt"
	"os"
	"path"

	"github.com/erkkah/git-secret/commands"
	"github.com/erkkah/git-secret/utils"
)

const errorExitCode = 126

func main() {
	args := os.Args
	if len(args) < 2 {
		usage("no input parameters provided")
	}

	err := checkSetup()

	if err == nil {
		cmd := args[1]
		err = runCommand(cmd, os.Args[2:])
	}
	if err != nil {
		fmt.Printf("%s: abort: %v\n", appName(), err)
		os.Exit(errorExitCode)
	}
}

func verifySecretsDirIsNotIgnored() error {
	secretsDir, err := utils.SecretsDir()
	if err != nil {
		return err
	}
	ignored, err := utils.IsGitIgnored(secretsDir)

	if err != nil {
		return err
	}

	if ignored {
		return fmt.Errorf("%q is in .gitignore", secretsDir)
	}

	return nil
}

func checkSetup() error {
	inTree, err := utils.IsInsideGitTree()
	if err != nil {
		return err
	}
	if !inTree {
		return fmt.Errorf("not in dir with git repo. Use 'git init' or 'git clone', then in repo use 'git secret init'")
	}

	err = verifySecretsDirIsNotIgnored()
	if err != nil {
		return err
	}

	secretsDir, err := utils.SecretsDir()
	if err != nil {
		return err
	}
	secRing := path.Join(secretsDir, "secring.gpg")
	stat, err := os.Stat(secRing)
	if err == nil && stat.Size() > 0 {
		return fmt.Errorf("it seems that someone has imported a secret key")
	}

	return nil
}

func appName() string {
	app := path.Base(os.Args[0])
	return app
}

func usage(message string) {
	fmt.Printf("%s: %s\n", appName(), message)
	fmt.Println("usage: ...")
	os.Exit(errorExitCode)
}

func runCommand(cmd string, args []string) error {
	cmds := map[string]func([]string) error{
		"init":   commands.Init,
		"add":    commands.Add,
		"remove": commands.Remove,
		"hide":   commands.Hide,
		"reveal": commands.Reveal,
		"keys":   commands.Keys,
	}
	command, found := cmds[cmd]
	if !found {
		return fmt.Errorf("command %q not found", cmd)
	}
	return command(args)
}
