package main

import (
	"fmt"
	"os"
	"path"

	"github.com/erkkah/git-private/commands"
	"github.com/erkkah/git-private/utils"
)

func main() {
	args := os.Args
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	err := checkSetup()

	if err == nil {
		cmd := args[1]
		err = runCommand(cmd, os.Args[2:])
	}
	if err != nil {
		fmt.Printf("%s: %v\n", appName(), err)
		os.Exit(1)
	}
}

func verifyStateDirIsNotIgnored() error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}
	ignored, err := utils.IsGitIgnored(stateDir)

	if err != nil {
		return err
	}

	if ignored {
		return fmt.Errorf("%q is in .gitignore", stateDir)
	}

	return nil
}

func checkSetup() error {
	inTree, err := utils.IsInsideGitTree()
	if err != nil {
		return err
	}
	if !inTree {
		return fmt.Errorf("not in dir with git repo. Use 'git init' or 'git clone', then in repo use 'git %s init'", utils.ToolName)
	}

	err = verifyStateDirIsNotIgnored()
	if err != nil {
		return err
	}

	return nil
}

func appName() string {
	app := path.Base(os.Args[0])
	return app
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
	%[1]s init
	%[1]s add <FILE...>
	%[1]s remove <FILE...>
	%[1]s hide [-keyfile FILE] [-clean] [FILE...]
	%[1]s reveal [-keyfile FILE] [-force] [FILE...]
	%[1]s keys list [-keyfile FILE]
	%[1]s keys add [-keyfile FILE] [-pubfile FILE] [-id ID] [public key]
	%[1]s keys remove [-keyfile FILE] [-id ID] [ID]
	%[1]s keys generate [-keyfile FILE] [-pubfile FILE]
	%[1]s clean
	%[1]s status

Example:
	$ git-private init
	$ git-private add apikey.txt
	$ git-private keys add -pubfile ~/.ssh/id_rsa.pub
	$ git-private hide -keyfile ~/.ssh/id_rsa
`, appName())
}

func runCommand(cmd string, args []string) error {
	cmds := map[string]func([]string, func()) error{
		"init":   commands.Init,
		"add":    commands.Add,
		"remove": commands.Remove,
		"hide":   commands.Hide,
		"reveal": commands.Reveal,
		"keys":   commands.Keys,
		"clean":  commands.Clean,
		"status": commands.Status,
		"help":   help,
	}
	command, found := cmds[cmd]
	if !found {
		return fmt.Errorf("command %q not found", cmd)
	}
	return command(args, usage)
}

func help(_ []string, usage func()) error {
	usage()
	return nil
}
