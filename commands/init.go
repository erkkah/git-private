package commands

import (
	"fmt"
	"os"

	"github.com/erkkah/git-secret/utils"
)

func Init(_ []string) error {
	secretsDir, err := utils.StateDir()
	if err != nil {
		return err
	}

	if utils.Exists(secretsDir) {
		return fmt.Errorf("already initialized")
	}

	err = os.MkdirAll(secretsDir, 0770)
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

	err = utils.GitAddIgnorePattern(fmt.Sprintf("!*%s", utils.SecretsExtension))
	if err != nil {
		return err
	}

	return nil
}
