package commands

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"github.com/erkkah/git-secret/utils"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func Reveal(args []string) error {
	var config struct {
		KeyFromEnv  string
		KeyFromFile string
	}

	flags := flag.NewFlagSet("reveal", flag.ExitOnError)
	flags.StringVar(&config.KeyFromEnv, "env", "", "Load private key from environment variable")
	flags.StringVar(&config.KeyFromFile, "file", "", "Load private key from file")
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

	filesToReveal := flags.Args()

	if len(filesToReveal) != 0 {
		for i := range filesToReveal {
			filesToReveal[i], err = utils.RepoRelative(filesToReveal[i])
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
			filesToReveal = append(filesToReveal, file.Path)
		}
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

	for _, file := range filesToReveal {
		if strings.HasSuffix(file, utils.SecretsExtension) {
			return fmt.Errorf("cannot decrypt to secret version of file:, %q", file)
		}

		// ??? check status and follow force flag

		err := decrypt(file, identity)
		if err != nil {
			return fmt.Errorf("decryption failed: %w", err)
		}
	}

	return nil
}

func decrypt(file string, identity age.Identity) error {
	root, err := utils.GetGitRootPath()
	if err != nil {
		return err
	}

	fullPath := path.Join(root, file)
	secretPath := fullPath + utils.SecretsExtension

	encrypted, err := os.Open(secretPath)
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
