package commands

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/erkkah/git-private/utils"
)

func Status(_ []string, _ func()) error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}
	exists, err := utils.Exists(stateDir)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s not initialized in repo", utils.ToolName)
	}

	files, err := utils.LoadFileList()
	if err != nil {
		return fmt.Errorf("failed to load file list: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	if len(files.Files) == 0 {
		fmt.Fprintln(w, "No private files")
	}
	for _, file := range files.Files {
		status, err := getFileStatus(file)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "%s\t[%s]\n", file.Path, status)
	}
	w.Flush()

	return nil
}

func areFilesInSync() (bool, error) {
	files, err := utils.LoadFileList()
	if err != nil {
		return false, err
	}

	for _, file := range files.Files {
		status, err := getFileStatus(file)
		if err != nil {
			return false, err
		}

		if status != hiddenInSync {
			return false, nil
		}
	}

	return true, nil
}

func getFileStatus(file utils.SecureFile) (statusCode, error) {
	root, err := utils.GetGitRootPath()
	if err != nil {
		return 0, err
	}
	fullPath := root.Join(file.Path)

	var status statusCode

	if file.Hash == "" {
		status = notHidden
	} else {
		privateFile := fullPath + utils.PrivateExtension
		if exists, err := utils.Exists(privateFile); !exists || err != nil {
			if err != nil {
				return 0, err
			}
			status = hiddenPrivateMissing
		} else if exists, err := utils.Exists(fullPath); !exists || err != nil {
			if err != nil {
				return 0, err
			}
			status = hiddenNotRevealed
		} else {
			hash, err := utils.GetFileHash(file.Path)
			if err != nil {
				return 0, err
			}
			if hash == file.Hash {
				status = hiddenInSync
			} else {
				status = hiddenModified
			}
		}
	}

	return status, nil
}

type statusCode int

const (
	notHidden statusCode = iota + 421
	hiddenInSync
	hiddenModified
	hiddenNotRevealed
	hiddenPrivateMissing
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
	case hiddenPrivateMissing:
		return "WARNING: private file missing!"
	default:
		return "unknown"
	}
}
