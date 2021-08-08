package commands

import (
	"flag"
	"fmt"
	"os"

	"github.com/erkkah/git-private/utils"
)

func Clean(args []string) error {
	var config struct {
		Force bool
	}

	flags := flag.NewFlagSet("hide [file]", flag.ExitOnError)
	flags.BoolVar(&config.Force, "force", false, "Force removal of out of sync files")
	flags.Parse(args)

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	err = cleanFiles(fileList.Files, config.Force)
	if err != nil {
		return err
	}

	return nil
}

func cleanFiles(filesToClean []utils.SecureFile, force bool) error {
	for _, file := range filesToClean {

		if !force {
			status, err := getFileStatus(file)
			if err != nil {
				return err
			}

			if status == hiddenPrivateMissing {
				return fmt.Errorf("will not remove file %q with missing private file, use 'force' flag to override", file.Path)
			}

			if status == hiddenModified {
				return fmt.Errorf("will not remove out of sync file %q, use 'force' flag to override", file.Path)
			}
		}

		err := os.Remove(file.Path)
		if err != nil {
			return fmt.Errorf("failed to remove file %q: %w", file.Path, err)
		}
	}

	return nil
}
