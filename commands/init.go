package commands

import (
	"fmt"
	"os"

	"github.com/erkkah/git-private/utils"
)

func Init(_ []string, _ func()) error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}

	exists, err := utils.Exists(stateDir)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("already initialized")
	}

	err = os.MkdirAll(stateDir.Absolute(), 0770)
	if err != nil {
		return err
	}

	err = utils.StoreFileList(utils.FileList{})
	if err != nil {
		return err
	}

	err = utils.GitAddIgnorePattern(fmt.Sprintf("!*%s", utils.PrivateExtension))
	if err != nil {
		return err
	}

	return nil
}
