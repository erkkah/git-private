package commands

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/erkkah/git-private/utils"
)

func Hide(args []string, usage func()) error {
	var config struct {
		KeyFromFile string
		Clean       bool
	}

	flags := flag.NewFlagSet("hide [file]", flag.ExitOnError)
	flags.StringVar(&config.KeyFromFile, "keyfile", "", "Load private key from `file`")
	flags.BoolVar(&config.Clean, "clean", false, "Remove source files after encryption")
	flags.Usage = usage
	flags.Parse(args)

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	var filesToHide []utils.RepoRelativePath

	fileArgs := flags.Args()
	if len(fileArgs) != 0 {
		for _, v := range fileArgs {
			absolute, err := filepath.Abs(v)
			if err != nil {
				return err
			}
			file, err := utils.RepoRelative(utils.AbsolutePath(absolute))
			if err != nil {
				return err
			}
			filesToHide = append(filesToHide, file)
		}
	} else {
		fileList, err := utils.LoadFileList()
		if err != nil {
			return err
		}

		for _, file := range fileList.Files {
			filesToHide = append(filesToHide, file.Path)
		}
	}

	identity, err := loadPrivateKey(config.KeyFromFile)
	if err != nil {
		return err
	}

	err = hideFiles(identity, filesToHide, config.Clean)
	if err != nil {
		return err
	}

	return nil
}

func hideFiles(identity age.Identity, filesToHide []utils.RepoRelativePath, clean bool) error {
	recipients, err := utils.GetRecipients(identity)
	if err != nil {
		return fmt.Errorf("failed to load keys, cannot encrypt: %w", err)
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no keys added, cannot encrypt")
	}

	for _, file := range filesToHide {
		if strings.HasSuffix(file.Relative(), utils.PrivateExtension) {
			return fmt.Errorf("cannot encrypt private file:, %q", file)
		}

		err := encrypt(file, recipients)
		if err != nil {
			return err
		}

		err = updateFileHash(file)
		if err != nil {
			return err
		}

		if clean {
			fullPath, err := utils.RepoAbsolute(file)
			if err != nil {
				return err
			}
			err = os.Remove(fullPath.Absolute())
			if err != nil {
				return fmt.Errorf("failed to remove source file after encryption: %w", err)
			}
		}
	}

	return nil
}

func encrypt(file utils.RepoRelativePath, recipients []age.Recipient) error {
	fullPath, err := utils.RepoAbsolute(file)
	if err != nil {
		return err
	}

	privatePath := fullPath.Absolute() + utils.PrivateExtension

	var buf bytes.Buffer
	encryptedWriter, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return err
	}

	privateFile, err := fullPath.Open()
	if err != nil {
		return err
	}

	_, err = io.Copy(encryptedWriter, privateFile)
	if err != nil {
		return err
	}

	encryptedWriter.Close()
	if err != nil {
		return err
	}

	err = os.WriteFile(privatePath, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	return nil
}

func updateFileHash(file utils.RepoRelativePath) error {
	hash, err := utils.GetFileHash(file)
	if err != nil {
		return err
	}

	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}
	for i, fileEntry := range fileList.Files {
		if fileEntry.Path == file {
			fileList.Files[i].Hash = hash
			err = utils.StoreFileList(fileList)
			return err
		}
	}

	return fmt.Errorf("file %q not in file list", file)
}
