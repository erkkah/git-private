package commands

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"filippo.io/age"
	"filippo.io/age/agessh"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"

	"github.com/erkkah/git-private/utils"
)

func Keys(args []string, usage func()) error {
	var config struct {
		PubKeyID   string
		PubKeyFile string
		KeyFile    string
		ReadOnly   bool
	}

	flags := flag.NewFlagSet("keys <list|add [key data]|remove|generate>", flag.ExitOnError)
	flags.StringVar(&config.PubKeyID, "id", "", "Key `identity` to add or remove")
	flags.StringVar(&config.PubKeyFile, "pubfile", "", "Load / store public key from / to `file`")
	flags.StringVar(&config.KeyFile, "keyfile", "", "Load / store private key from / to `file`")
	flags.BoolVar(&config.ReadOnly, "readonly", false, "Added key can only be used to reveal files")
	flags.Usage = usage

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		flags.Parse(args[1:])
	} else {
		return fmt.Errorf("no keys command specified, expected <list|add|remove|generate>")
	}

	cmd := args[0]

	if cmd != "generate" {
		err := utils.EnsureInitialized()
		if err != nil {
			return err
		}
	}

	var err error

	switch {
	case cmd == "list":
		identity, err := loadPrivateKey(config.KeyFile)
		if err != nil {
			return err
		}

		return listKeys(identity)

	case cmd == "add":
		var key string

		if config.PubKeyFile != "" {
			key, err = utils.ReadFromFileOrStdin(config.PubKeyFile)
			if err != nil {
				return fmt.Errorf("failed to load public key from %q: %w", config.PubKeyFile, err)
			}
		} else {
			key = flags.Arg(0)
			if key == "" {
				return fmt.Errorf("no public key specified")
			}
		}

		identity, err := loadPrivateKey(config.KeyFile)
		if err != nil {
			return err
		}

		access := utils.ReadWrite
		if config.ReadOnly {
			access = utils.ReadOnly
		}
		err = addKey(identity, config.PubKeyID, key, access)
		if err != nil {
			return err
		}

		inSync, err := areFilesInSync()
		if err != nil {
			return err
		}
		if inSync {
			err = reHideFiles(identity)
			if err != nil {
				return fmt.Errorf("failed to re-encrypt files after key addition")
			}
		} else {
			fmt.Fprintf(os.Stderr, "Files are not in sync, will not re-encrypt after key change. Use 'hide' and/or 'reveal' accordingly.\n")
		}

	case cmd == "remove":
		if config.PubKeyID == "" {
			config.PubKeyID = flags.Arg(0)
		}
		if config.PubKeyID == "" {
			return fmt.Errorf("specify identity of key to remove")
		}

		identity, err := loadPrivateKey(config.KeyFile)
		if err != nil {
			return err
		}

		err = removeKey(identity, config.PubKeyID)
		if err != nil {
			return err
		}
		err = reHideFiles(identity)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt files after key removal")
		}

	case cmd == "generate":
		if config.KeyFile == "" {
			return fmt.Errorf("use 'keyfile' flag to specify target file for generated key")
		}
		exists, err := utils.Exists(utils.AbsolutePath(config.KeyFile))
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("will not overwrite existing key file %q", config.KeyFile)
		}

		generated, err := age.GenerateX25519Identity()
		if err != nil {
			return err
		}

		passphrase, err := readPassphrase("Enter passphrase:")
		if err != nil {
			return err
		}

		if len(passphrase) != 0 {
			confirmed, err := readPassphrase("Confirm passphrase:")
			if err != nil {
				return err
			}

			if !bytes.Equal(passphrase, confirmed) {
				return fmt.Errorf("passphrases do not match")
			}
		}

		return exportGeneratedKey(generated, config.KeyFile, config.PubKeyFile, passphrase)

	default:
		return fmt.Errorf("unknown keys command %q", cmd)
	}

	return nil
}

func reHideFiles(identity age.Identity) error {
	fileList, err := utils.LoadFileList()
	if err != nil {
		return err
	}

	var paths []utils.RepoRelativePath
	for _, file := range fileList.Files {
		paths = append(paths, file.Path)
	}

	return hideFiles(identity, paths, false)
}

func listKeys(identity age.Identity) error {
	keyList, err := utils.LoadKeyList(identity)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	for _, key := range keyList.Keys {
		modeString := "rw"
		if key.ReadOnly {
			modeString = "ro"
		}
		fmt.Fprintf(w, "%s\t(%s/%s)\t[...%s]\n", key.ID, key.Type, modeString, key.Key[len(key.Key)-12:])
	}
	w.Flush()
	return nil
}

func addKey(identity age.Identity, id string, key string, access utils.KeyAccess) error {
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
		return storeKey(identity, utils.SSH, id, keyString, access)
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
	return storeKey(identity, utils.AGE, id, key, access)
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

func storeKey(identity age.Identity, keyType utils.KeyType, id string, keyData string, access utils.KeyAccess) error {
	keyList, err := utils.LoadKeyList(identity)
	if err != nil {
		return err
	}

	for _, key := range keyList.Keys {
		if key.ID == id {
			return fmt.Errorf("key with id %q already exists", id)
		}
	}

	readOnly := access == utils.ReadOnly
	keyList.Keys = append(keyList.Keys, utils.Key{
		Type:     keyType,
		ID:       id,
		Key:      keyData,
		ReadOnly: readOnly,
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

			keyData, err := os.ReadFile(keyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read private key file %q: %w", keyFile, err)
			}

			key = string(keyData)
		}
	}

	identity, err := parseSSHIdentity([]byte(key))
	if err != nil {
		identity, err = parseAGEIdentity([]byte(key))
	}
	if err != nil {
		return nil, err
	}

	return identity, nil
}

func parseSSHIdentity(key []byte) (age.Identity, error) {
	identity, err := agessh.ParseIdentity(key)
	if err != nil {
		if _, needsPassword := err.(*ssh.PassphraseMissingError); needsPassword {
			passphrase, err := readPassphrase("Enter SSH key passphrase:")
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
		} else {
			return nil, err
		}
	}

	return identity, nil
}

func parseAGEIdentity(key []byte) (age.Identity, error) {
	var ageMagic = []byte("age-encryption.org/")

	if bytes.HasPrefix(key, ageMagic) {
		passphrase, err := readPassphrase("Enter AGE key passphrase:")
		if err != nil {
			return nil, fmt.Errorf("failed to read passphrase")
		}

		identity, err := age.NewScryptIdentity(string(passphrase))
		if err != nil {
			return nil, err
		}

		reader := bytes.NewReader(key)
		decryptor, err := age.Decrypt(reader, identity)
		if err != nil {
			return nil, err
		}

		var decrypted bytes.Buffer
		_, err = io.Copy(&decrypted, decryptor)
		if err != nil {
			return nil, err
		}

		key = decrypted.Bytes()
	}

	if parsedIdentities, err := age.ParseIdentities(bytes.NewReader(key)); err == nil && len(parsedIdentities) == 1 {
		return parsedIdentities[0], nil
	}

	return nil, fmt.Errorf("invalid key or passphase")
}

func readPassphrase(prompt string) ([]byte, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	stdinFd := os.Stdin.Fd()
	state, _ := term.GetState(int(stdinFd))
	go func() {
		signal := <-signals
		if signal != nil && state != nil {
			term.Restore(int(stdinFd), state)
			os.Exit(1)
		}
	}()
	defer func() {
		close(signals)
		signal.Reset(syscall.SIGINT)
	}()

	fmt.Print(prompt)
	passphrase, err := term.ReadPassword(int(stdinFd))
	fmt.Println()

	return passphrase, err
}

func exportGeneratedKey(key *age.X25519Identity, keyFile string, pubKeyFile string, passphrase []byte) error {
	var target io.WriteCloser
	target, err := os.Create(keyFile)
	if err != nil {
		return err
	}

	cleanup := func(err error) error {
		_ = target.Close()
		_ = os.Remove(keyFile)
		return err
	}

	public := key.Recipient().String()
	private := key.String()

	if len(passphrase) == 0 {
		fmt.Fprintln(os.Stderr, "no passphrase given, generated key will be stored in clear text")
	} else {
		passPhraseRecipient, err := age.NewScryptRecipient(string(passphrase))
		if err != nil {
			return cleanup(err)
		}

		target, err = age.Encrypt(target, passPhraseRecipient)
		if err != nil {
			return cleanup(err)
		}

		publicKeyString := fmt.Sprintf("Public key: %v\n", public)
		if pubKeyFile != "" {
			err = os.WriteFile(pubKeyFile, []byte(publicKeyString), 0600)
			if err != nil {
				return cleanup(fmt.Errorf("failed to write public key file: %w", err))
			}
		} else {
			fmt.Fprint(os.Stderr, publicKeyString)
		}
	}

	timestamp := time.Now().Format(time.RFC3339)
	_, err = fmt.Fprintf(target, "# created: %v\n# public key: %v\n%v\n", timestamp, public, private)
	if err != nil {
		return cleanup(err)
	}

	err = target.Close()
	if err != nil {
		return cleanup(err)
	}

	return nil
}
