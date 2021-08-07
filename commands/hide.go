package commands

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"

	"github.com/erkkah/git-private/utils"
)

func Hide(args []string) error {
	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	var config struct {
		Clean bool
	}

	flags := flag.NewFlagSet("hide [file]", flag.ExitOnError)
	flags.BoolVar(&config.Clean, "clean", false, "Remove source files after encryption")
	flags.Parse(args)

	filesToHide := flags.Args()

	if len(filesToHide) != 0 {
		for i := range filesToHide {
			filesToHide[i], err = utils.RepoRelative(filesToHide[i])
			if err != nil {
				return err
			}
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

	for _, file := range filesToHide {
		if strings.HasSuffix(file, utils.PrivateExtension) {
			return fmt.Errorf("cannot encrypt private file:, %q", file)
		}

		err := encrypt(file)
		if err != nil {
			return err
		}

		err = updateFileHash(file)
		if err != nil {
			return err
		}

		if config.Clean {
			fullPath, err := utils.RepoAbsolute(file)
			if err != nil {
				return err
			}
			err = os.Remove(fullPath)
			if err != nil {
				return fmt.Errorf("failed to remove source file after encryption: %w", err)
			}
		}
	}

	return nil
}

func encrypt(file string) error {
	fullPath, err := utils.RepoAbsolute(file)
	if err != nil {
		return err
	}

	privatePath := fullPath + utils.PrivateExtension

	recipients, err := getRecipients()
	if len(recipients) == 0 {
		return fmt.Errorf("no keys added, cannot encrypt")
	}

	buf := bytes.NewBuffer([]byte{})
	encryptedWriter, err := age.Encrypt(buf, recipients...)
	if err != nil {
		return err
	}

	privateFile, err := os.Open(fullPath)
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

	err = ioutil.WriteFile(privatePath, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	return nil
}

func getRecipients() ([]age.Recipient, error) {
	keyList, err := utils.LoadKeyList()
	if err != nil {
		return nil, err
	}

	var recipients []age.Recipient

	for _, key := range keyList.Keys {
		var recipient age.Recipient

		if key.Type == utils.AGE {
			parsedRecipients, err := age.ParseRecipients(strings.NewReader(key.Key))
			if err != nil {
				return nil, err
			}
			if len(parsedRecipients) != 1 {
				return nil, fmt.Errorf("unexpected key contents")
			}
			recipient = parsedRecipients[0]
		} else if key.Type == utils.SSH {
			recipient, err = agessh.ParseRecipient(key.Key)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("unexpected key type %q", key.Type)
		}

		recipients = append(recipients, recipient)
	}

	return recipients, nil
}

func updateFileHash(file string) error {
	hash, err := utils.GetFileHash(file)
	if err != nil {
		return err
	}

	fileList, err := utils.LoadFileList()
	for i, fileEntry := range fileList.Files {
		if fileEntry.Path == file {
			fileList.Files[i].Hash = hash
			err = utils.StoreFileList(fileList)
			return err
		}
	}

	return fmt.Errorf("file %q not in file list", file)
}
