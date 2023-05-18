package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/erkkah/git-private/utils"
)

func Remove(files []string, _ func()) error {
	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to remove")
	}

	var filesToRemove []utils.RepoRelativePath

	for _, file := range files {
		absolute, err := filepath.Abs(file)
		if err != nil {
			return err
		}
		repoRelative, err := utils.RepoRelative(utils.AbsolutePath(absolute))
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

func removeFiles(files []utils.RepoRelativePath) error {
	for _, file := range files {
		err := utils.GitRemoveIgnorePattern(file.Relative())
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

func removeFile(file utils.RepoRelativePath) error {
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

	privateFile, err := utils.RepoAbsolute(file + utils.PrivateExtension)
	if err != nil {
		return err
	}

	exists, err := utils.Exists(privateFile)
	if err != nil {
		return err
	}
	if exists {
		err := os.Remove(privateFile.Absolute())
		if err != nil {
			return err
		}
	}

	return nil
}
