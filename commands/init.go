package commands

import (
	"fmt"
	"os"

	"github.com/erkkah/git-private/utils"
)

func Init(_ []string) error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}

	if utils.Exists(stateDir) {
		return fmt.Errorf("already initialized")
	}

	err = os.MkdirAll(stateDir, 0770)
	if err != nil {
		return err
	}

	err = utils.StoreFileList(utils.FileList{})
	if err != nil {
		return err
	}

	err = utils.StoreKeyList(utils.KeyList{})
	if err != nil {
		return err
	}

	err = utils.GitAddIgnorePattern(fmt.Sprintf("!*%s", utils.PrivateExtension))
	if err != nil {
		return err
	}

	return nil
}
