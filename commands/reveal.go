package commands

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/erkkah/git-private/utils"
)

func Reveal(args []string) error {
	var config struct {
		KeyFromEnv  string
		KeyFromFile string
		Overwrite   bool
	}

	flags := flag.NewFlagSet("reveal", flag.ExitOnError)
	flags.StringVar(&config.KeyFromEnv, "env", "", "Load private key from environment `variable`")
	flags.StringVar(&config.KeyFromFile, "file", "", "Load private key from file")
	flags.BoolVar(&config.Overwrite, "force", false, "Overwrite existing target files")
	flags.Parse(args)

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	var key string
	if config.KeyFromEnv != "" {
		key = os.Getenv(config.KeyFromEnv)
	} else if config.KeyFromFile != "" {
		key, err = utils.ReadFromFileOrStdin(config.KeyFromFile)
		if err != nil {
			return fmt.Errorf("failed to load key from %q: %w", config.KeyFromFile, err)
		}
	} else {
		return fmt.Errorf("use '-env' or '-file' to specify key source")
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

	identity, err := agessh.ParseIdentity([]byte(key))
	if err != nil {
		if _, needsPassword := err.(*ssh.PassphraseMissingError); needsPassword {
			fmt.Print("Enter passphrase:")
			passphrase, err := terminal.ReadPassword(0)
			if err != nil {
				return fmt.Errorf("failed to read passphrase")
			}
			parsedIdentity, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(key), passphrase)
			if err != nil {
				return fmt.Errorf("failed to load key, wrong passphrase?")
			}

			switch k := parsedIdentity.(type) {
			case *ed25519.PrivateKey:
				identity, err = agessh.NewEd25519Identity(*k)
			case *rsa.PrivateKey:
				identity, err = agessh.NewRSAIdentity(k)
			default:
				err = fmt.Errorf("unsupported SSH key type: %T", k)
			}

			if err != nil {
				return err
			}
		} else if parsedIdentities, err := age.ParseIdentities(strings.NewReader(key)); err == nil && len(parsedIdentities) == 1 {
			identity = parsedIdentities[0]
		} else {
			return fmt.Errorf("invalid key")
		}
	}

	revealed := 0
	for _, file := range filesToReveal {
		status, err := getFileStatus(file)
		if err != nil {
			return fmt.Errorf("failed to get file status: %w", err)
		}
		switch status {
		case hiddenInSync:
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
		err = decrypt(file.Path, identity)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
		revealed++
	}

	suffix := "s"
	if revealed == 1 {
		suffix = ""
	}
	fmt.Printf("%v file%s revealed\n", revealed, suffix)

	return nil
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

func decrypt(file string, identity age.Identity) error {
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

	buf := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buf, decryptedReader)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fullPath, buf.Bytes(), 0660)
	if err != nil {
		return err
	}

	return nil
}
