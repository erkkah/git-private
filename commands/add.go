package commands

import (
	"fmt"

	"github.com/erkkah/git-private/utils"
)

func Add(files []string, _ func()) error {
	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return fmt.Errorf("no files to add")
	}

	var filesToAdd []string

	for _, file := range files {
		exists, err := utils.Exists(file)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no such file: %q", file)
		}
		repoRelative, err := utils.RepoRelative(file)
		if err != nil {
			return err
		}
		filesToAdd = append(filesToAdd, repoRelative)
	}

	err = addFiles(filesToAdd)
	if err != nil {
		return err
	}

	return nil
}

func addFiles(files []string) error {
	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	for _, file := range files {
		if !hasFile(fileList, file) {
			fileList.Files = append(fileList.Files, utils.SecureFile{
				Path: file,
			})
			err = utils.GitAddIgnorePattern(file)
			if err != nil {
				return err
			}
		}
	}

	err = utils.StoreFileList(fileList)
	if err != nil {
		return err
	}

	return nil
}

func hasFile(fileList utils.FileList, file string) bool {
	for _, fileEntry := range fileList.Files {
		if fileEntry.Path == file {
			return true
		}
	}
	return false
}
