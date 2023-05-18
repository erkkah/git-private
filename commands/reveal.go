package commands

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"

	"filippo.io/age"

	"github.com/erkkah/git-private/utils"
)

func Reveal(args []string, usage func()) error {
	var config struct {
		KeyFromFile string
		Overwrite   bool
		Clean       bool
	}

	flags := flag.NewFlagSet("reveal", flag.ExitOnError)
	flags.StringVar(&config.KeyFromFile, "keyfile", "", "Load private key from `file`")
	flags.BoolVar(&config.Overwrite, "force", false, "Overwrite existing target files")
	flags.BoolVar(&config.Clean, "clean", false, "Remove private files after revealing")
	flags.Usage = usage
	flags.Parse(args)

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	var filesToReveal []utils.SecureFile
	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	if len(flags.Args()) != 0 {
		for _, arg := range flags.Args() {
			file, err := findFile(arg, fileList.Files)
			if err == errNotFound {
				return fmt.Errorf("file %q is not hidden", arg)
			}
			if err != nil {
				return fmt.Errorf("failed to look up file: %w", err)
			}
			filesToReveal = append(filesToReveal, file)
		}
	} else {
		filesToReveal = fileList.Files
	}

	identity, err := loadPrivateKey(config.KeyFromFile)
	if err != nil {
		return err
	}

	revealed := 0
	inSync := 0

	for _, file := range filesToReveal {
		status, err := getFileStatus(file)
		if err != nil {
			return fmt.Errorf("failed to get file status: %w", err)
		}
		switch status {
		case hiddenInSync:
			inSync++
			continue
		case hiddenPrivateMissing:
			return fmt.Errorf("cannot reveal, private version of %q is missing", file.Path)
		case hiddenModified:
			if !config.Overwrite {
				return fmt.Errorf("will not overwrite existing file %q without 'force' flag", file.Path)
			}
		case notHidden:
			return fmt.Errorf("file %q is not hidden", file.Path)
		case hiddenNotRevealed:
		}
		err = decrypt(file.Path, config.Clean, identity)
		if err != nil {
			return fmt.Errorf("reveal failed: %w", err)
		}
		revealed++
	}

	if inSync > 0 {
		fmt.Printf("%v file%s already in sync, ", inSync, pluralSuffix(inSync))
	}
	fmt.Printf("%v file%s revealed\n", revealed, pluralSuffix(revealed))

	return nil
}

func pluralSuffix(number int) string {
	suffix := "s"
	if number == 1 {
		suffix = ""
	}
	return suffix
}

var errNotFound = errors.New("not found")

func findFile(path string, files []utils.SecureFile) (utils.SecureFile, error) {
	repoPath, err := utils.RepoRelative(path)
	if err != nil {
		return utils.SecureFile{}, err
	}

	for _, file := range files {
		if file.Path == repoPath {
			return file, nil
		}
	}

	return utils.SecureFile{}, errNotFound
}

func decrypt(file string, clean bool, identity age.Identity) error {
	root, err := utils.GetGitRootPath()
	if err != nil {
		return err
	}

	fullPath := path.Join(root, file)
	privatePath := fullPath + utils.PrivateExtension

	encrypted, err := os.Open(privatePath)
	if err != nil {
		return err
	}

	decryptedReader, err := age.Decrypt(encrypted, identity)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, decryptedReader)
	if err != nil {
		return err
	}

	err = os.WriteFile(fullPath, buf.Bytes(), 0660)
	if err != nil {
		return err
	}

	if clean {
		err = os.Remove(privatePath)
		if err != nil {
			return fmt.Errorf("Revealed, but failed to remove private file: %w", err)
		}
	}

	return nil
}
