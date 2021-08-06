package commands

import (
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	"github.com/erkkah/git-private/utils"
)

func Status(_ []string) error {
	stateDir, err := utils.StateDir()
	if err != nil {
		return err
	}
	if !utils.Exists(stateDir) {
		return fmt.Errorf("%s not initialized in repo", utils.ToolName)
	}

	files, err := utils.LoadFileList()
	if err != nil {
		return fmt.Errorf("failed to load file list: %w", err)
	}

	root, err := utils.GetGitRootPath()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	for _, file := range files.Files {
		fullPath := path.Join(root, file.Path)

		var status statusCode

		if file.Hash == "" {
			status = notHidden
		} else {
			privateFile := fullPath + utils.PrivateExtension
			if !utils.Exists(privateFile) {
				status = hiddenPrivateMissing
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

		fmt.Fprintf(w, "%s\t[%s]\n", file.Path, status)
	}
	w.Flush()

	return nil
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
