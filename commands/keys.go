package commands

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/erkkah/git-secret/utils"
	"golang.org/x/crypto/ssh"
)

func Keys(args []string) error {
	var config struct {
		Identity    string
		KeyFromEnv  string
		KeyFromFile string
	}

	flags := flag.NewFlagSet("keys", flag.ExitOnError)
	flags.StringVar(&config.Identity, "id", "", "Key identity")
	flags.StringVar(&config.KeyFromEnv, "env", "", "Load public key from environment variable")
	flags.StringVar(&config.KeyFromFile, "file", "-", "Load public key from file (default stdin)")
	if len(args) > 1 {
		flags.Parse(args[1:])
	}

	err := utils.EnsureInitialized()
	if err != nil {
		return err
	}

	cmd := "list"

	if len(args) > 0 {
		cmd = args[0]
	}

	switch {
	case cmd == "" || cmd == "list":
		return listKeys()
	case cmd == "add":
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
		return addKey(config.Identity, key)
	case cmd == "remove":
		if config.Identity == "" {
			return fmt.Errorf("specify identity of key to remove")
		}
		return removeKey(config.Identity)
	default:
		return fmt.Errorf("unknown keys command %q", cmd)
	}
}

func listKeys() error {
	keyList, err := utils.LoadKeyList()
	if err != nil {
		return err
	}
	fmt.Println("Keys:")
	for _, key := range keyList.Keys {
		fmt.Printf("ID: %s, Type: %s\n", key.ID, key.Type)
	}
	return nil
}

func addKey(id string, key string) error {
	sshKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(id))
	if err == nil {
		if id == "" {
			id = strings.TrimSpace(comment)
		}
		if id == "" {
			return fmt.Errorf("key has no comment, and no id specified")
		}
		/*
			recipient, err := agessh.NewRSARecipient(sshKey)
			if err != nil {
				return err
			}
		*/
		keyData := ssh.MarshalAuthorizedKey(sshKey)
		return storeKey(utils.SSH, id, string(keyData))
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
	return storeKey(utils.AGE, id, key)
}

func removeKey(id string) error {
	keyList, err := utils.LoadKeyList()
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

	err = utils.StoreKeyList(updatedList)
	if err != nil {
		return err
	}

	return nil
}

func storeKey(keyType utils.KeyType, id string, keyData string) error {
	keyList, err := utils.LoadKeyList()
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

	err = utils.StoreKeyList(keyList)
	if err != nil {
		return err
	}
	return nil
}
