package commands

import (
	"fmt"
	"path"

	"github.com/erkkah/git-secret/utils"
)

func Status(_ []string) error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}
	if !utils.Exists(stateDir) {
		return fmt.Errorf("git-private not initialized in repo")
	}

	files, err := utils.LoadFileList()
	if err != nil {
		return fmt.Errorf("failed to load file list: %w", err)
	}

	root, err := utils.GetGitRootPath()
	if err != nil {
		return err
	}

	for _, file := range files.Files {
		fullPath := path.Join(root, file.Path)

		var status statusCode

		if file.Hash == "" {
			status = notHidden
		} else {
			privateFile := fullPath + utils.SecretsExtension
			if !utils.Exists(privateFile) {
				status = hiddenSecretMissing
			} else if !utils.Exists(fullPath) {
				status = hiddenNotRevealed
			} else {
				hash, err := utils.GetFileHash(fullPath)
				if err != nil {
					return err
				}
				if hash == file.Hash {
					status = hiddenInSync
				} else {
					status = hiddenModified
				}
			}
		}

		fmt.Printf("%s\t-\t%s\n", file.Path, status)
	}
	return nil
}

type statusCode int

const (
	notHidden statusCode = iota + 421
	hiddenInSync
	hiddenModified
	hiddenNotRevealed
	hiddenSecretMissing
)

func (code statusCode) String() string {
	switch code {
	case notHidden:
		return "not hidden"
	case hiddenInSync:
		return "hidden, in sync"
	case hiddenModified:
		return "hidden, modified"
	case hiddenNotRevealed:
		return "hidden, not revealed"
	case hiddenSecretMissing:
		return "WARNING: private file missing!"
	default:
		return "unknown"
	}
}
