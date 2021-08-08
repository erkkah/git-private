package commands

import (
	"crypto/ed25519"
	"crypto/rsa"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/tabwriter"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/erkkah/git-private/utils"
)

func Keys(args []string) error {
	var config struct {
		PubKeyID       string
		PubKeyFromEnv  string
		PubKeyFromFile string
		KeyFromFile    string
	}

	flags := flag.NewFlagSet("keys <list|add [key data]|remove>", flag.ExitOnError)
	flags.StringVar(&config.PubKeyID, "id", "", "Key `identity` to add or remove")
	flags.StringVar(&config.PubKeyFromEnv, "pubenv", "", "Load public key from environment `variable`")
	flags.StringVar(&config.PubKeyFromFile, "pubfile", "", "Load public key from `file`")
	flags.StringVar(&config.KeyFromFile, "keyfile", "", "Load private key from `file`")

	if len(args) > 1 && !strings.HasPrefix(args[0], "-") {
		flags.Parse(args[1:])
	} else {
		return fmt.Errorf("no keys command specified, expected <list|add|remove>")
	}

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	identity, err := loadPrivateKey(config.KeyFromFile)
	if err != nil {
		return err
	}

	cmd := "list"

	if len(args) > 0 {
		cmd = args[0]
	}

	switch {
	case cmd == "list":
		return listKeys(identity)
	case cmd == "add":
		var key string

		if config.PubKeyFromEnv != "" {
			key = os.Getenv(config.PubKeyFromEnv)
		} else if config.PubKeyFromFile != "" {
			key, err = utils.ReadFromFileOrStdin(config.PubKeyFromFile)
			if err != nil {
				return fmt.Errorf("failed to load public key from %q: %w", config.PubKeyFromFile, err)
			}
		} else {
			key = flags.Arg(0)
			if key == "" {
				return fmt.Errorf("no public key specified")
			}
		}
		return addKey(identity, config.PubKeyID, key)
	case cmd == "remove":
		if config.PubKeyID == "" {
			config.PubKeyID = flags.Arg(0)
		}
		if config.PubKeyID == "" {
			return fmt.Errorf("specify identity of key to remove")
		}
		return removeKey(identity, config.PubKeyID)
	default:
		return fmt.Errorf("unknown keys command %q", cmd)
	}
}

func listKeys(identity age.Identity) error {
	keyList, err := utils.LoadKeyList(identity)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	for _, key := range keyList.Keys {
		fmt.Fprintf(w, "%s\t[%s]\n", key.ID, key.Type)
	}
	w.Flush()
	return nil
}

func addKey(identity age.Identity, id string, key string) error {
	sshKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
	if err == nil {
		if id == "" {
			id = strings.TrimSpace(comment)
		}
		if id == "" {
			return fmt.Errorf("key has no comment, and no id specified")
		}
		keyData := ssh.MarshalAuthorizedKey(sshKey)
		keyString := strings.TrimSpace(string(keyData))
		return storeKey(identity, utils.SSH, id, keyString)
	}

	recipients, err := age.ParseRecipients(strings.NewReader(key))
	if err != nil {
		return fmt.Errorf("invalid key format")
	}
	if len(recipients) > 1 {
		return fmt.Errorf("multiple keys found, add one key at a time")
	}
	if len(recipients) == 0 {
		return fmt.Errorf("invalid key format")
	}
	if id == "" {
		return fmt.Errorf("cannot add AGE key without id")
	}
	return storeKey(identity, utils.AGE, id, key)
}

func removeKey(identity age.Identity, id string) error {
	keyList, err := utils.LoadKeyList(identity)
	if err != nil {
		return err
	}

	var updatedList utils.KeyList
	for _, key := range keyList.Keys {
		if key.ID == id {
			continue
		}
		updatedList.Keys = append(updatedList.Keys, key)
	}

	if len(updatedList.Keys) == len(keyList.Keys) {
		return fmt.Errorf("key %q not found", id)
	}

	err = utils.StoreKeyList(identity, updatedList)
	if err != nil {
		return err
	}

	return nil
}

func storeKey(identity age.Identity, keyType utils.KeyType, id string, keyData string) error {
	keyList, err := utils.LoadKeyList(identity)
	if err != nil {
		return err
	}

	for _, key := range keyList.Keys {
		if key.ID == id {
			return fmt.Errorf("key with id %q already exists", id)
		}
	}

	keyList.Keys = append(keyList.Keys, utils.Key{
		Type: keyType,
		ID:   id,
		Key:  keyData,
	})

	err = utils.StoreKeyList(identity, keyList)
	if err != nil {
		return err
	}
	return nil
}

func loadPrivateKey(loadFromFile string) (age.Identity, error) {
	var key string
	var err error

	if loadFromFile != "" {
		key, err = utils.ReadFromFileOrStdin(loadFromFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load key from %q: %w", loadFromFile, err)
		}
	} else {
		key = os.Getenv(utils.PrivateKeyVariable)
		if key == "" {
			keyFile := os.Getenv(utils.PrivateKeyFileVariable)
			if keyFile == "" {
				return nil, fmt.Errorf("no private key provided, use -keyfile or environment variables %s or %s",
					utils.PrivateKeyVariable, utils.PrivateKeyFileVariable)
			}

			keyData, err := ioutil.ReadFile(keyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file %q: %w", keyFile, err)
			}

			key = string(keyData)
		}
	}

	identity, err := agessh.ParseIdentity([]byte(key))
	if err != nil {
		if _, needsPassword := err.(*ssh.PassphraseMissingError); needsPassword {
			fmt.Print("Enter passphrase:")
			passphrase, err := terminal.ReadPassword(0)
			if err != nil {
				return nil, fmt.Errorf("failed to read passphrase")
			}
			parsedIdentity, err := ssh.ParseRawPrivateKeyWithPassphrase([]byte(key), passphrase)
			if err != nil {
				return nil, fmt.Errorf("failed to load key, wrong passphrase?")
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
				return nil, err
			}
		} else if parsedIdentities, err := age.ParseIdentities(strings.NewReader(key)); err == nil && len(parsedIdentities) == 1 {
			identity = parsedIdentities[0]
		} else {
			return nil, fmt.Errorf("invalid key")
		}
	}

	return identity, nil
}
