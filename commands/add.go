package commands

import (
	"fmt"
	"path/filepath"

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

	var filesToAdd []utils.RepoRelativePath

	for _, file := range files {
		if !filepath.IsAbs(file) {
			file, err = filepath.Abs(file)
			if err != nil {
				return err
			}
		}

		absolute := utils.AbsolutePath(file)
		exists, err := utils.Exists(absolute)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("no such file: %q", file)
		}
		repoRelative, err := utils.RepoRelative(absolute)
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

func addFiles(files []utils.RepoRelativePath) error {
	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	for _, file := range files {
		if !hasFile(fileList, file) {
			fileList.Files = append(fileList.Files, utils.SecureFile{
				Path: file,
			})
			err = utils.GitAddIgnorePattern(file.Relative())
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

func hasFile(fileList utils.FileList, file utils.RepoRelativePath) bool {
	for _, fileEntry := range fileList.Files {
		if fileEntry.Path == file {
			return true
		}
	}
	return false
}
