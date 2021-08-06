package commands

import (
	"fmt"
	"os"

	"github.com/erkkah/git-private/utils"
)

func Remove(files []string) error {
	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to remove")
	}

	var filesToRemove []string

	for _, file := range files {
		repoRelative, err := utils.RepoRelative(file)
		if err != nil {
			return err
		}
		filesToRemove = append(filesToRemove, repoRelative)
	}

	err = removeFiles(filesToRemove)
	if err != nil {
		return err
	}

	return nil
}

func removeFiles(files []string) error {
	for _, file := range files {
		err := utils.GitRemoveIgnorePattern(file)
		if err != nil {
			return err
		}
		err = removeFile(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeFile(file string) error {
	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	var updatedList utils.FileList
	for _, fileEntry := range fileList.Files {
		if fileEntry.Path != file {
			updatedList.Files = append(updatedList.Files, fileEntry)
		}
	}

	err = utils.StoreFileList(updatedList)
	if err != nil {
		return err
	}

	privateFile := file + utils.PrivateExtension
	if utils.Exists(privateFile) {
		// revealFile(file) ???
		err := os.Remove(privateFile)
		if err != nil {
			return err
		}
	}

	return nil
}
