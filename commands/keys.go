package commands

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base32"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

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
		KeyFile        string
	}

	flags := flag.NewFlagSet("keys <list|add [key data]|remove|generate>", flag.ExitOnError)
	flags.StringVar(&config.PubKeyID, "id", "", "Key `identity` to add or remove")
	flags.StringVar(&config.PubKeyFromEnv, "pubenv", "", "Load public key from environment `variable`")
	flags.StringVar(&config.PubKeyFromFile, "pubfile", "", "Load public key from `file`")
	flags.StringVar(&config.KeyFile, "keyfile", "", "Load / store private key from / to `file`")

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		flags.Parse(args[1:])
	} else {
		return fmt.Errorf("no keys command specified, expected <list|add|remove|generate>")
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
	case cmd == "list":
		identity, err := loadPrivateKey(config.KeyFile)
		if err != nil {
			return err
		}

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

		identity, err := loadPrivateKey(config.KeyFile)
		if err != nil {
			return err
		}

		err = addKey(identity, config.PubKeyID, key)
		if err != nil {
			return err
		}
		err = reHideFiles(identity)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt files after key addition")
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
		if utils.Exists(config.KeyFile) {
			return fmt.Errorf("will not overwrite existing key file %q", config.KeyFile)
		}

		generated, err := age.GenerateX25519Identity()
		if err != nil {
			return err
		}

		/*
			passphrase, err := readPassphrase()
			if err != nil {
				return err
			}
		*/

		passphrase := []byte("asdf")

		return exportGeneratedKey(generated, config.KeyFile, passphrase)

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

	var paths []string
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
		fmt.Fprintf(w, "%s\t[%s]\t(...%s)\n", key.ID, key.Type, key.Key[len(key.Key)-12:])
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
			passphrase, err := readPassphrase()
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
		} else if parsedIdentity, err := parseProtectedKey(key); err == nil {
			identity = parsedIdentity
		} else if parsedIdentities, err := age.ParseIdentities(strings.NewReader(key)); err == nil && len(parsedIdentities) == 1 {
			identity = parsedIdentities[0]
		} else {
			return nil, fmt.Errorf("invalid key")
		}
	}

	return identity, nil
}

func readPassphrase() ([]byte, error) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	state, _ := terminal.GetState(syscall.Stdin)
	go func() {
		select {
		case signal := <-signals:
			if signal != nil && state != nil {
				terminal.Restore(syscall.Stdin, state)
				os.Exit(1)
			}
		}
	}()
	defer func() {
		close(signals)
		signal.Reset(syscall.SIGINT)
	}()

	fmt.Print("Enter passphrase:")
	passphrase, err := terminal.ReadPassword(syscall.Stdin)
	fmt.Println()

	return passphrase, err
}

const secretKeyHRP = "AGE-SECRET-KEY-"
const protectedKeyHRP = "GIT-PRIVATE-PROTECTED-KEY-"

func parseProtectedKey(s string) (age.Identity, error) {
	lines := strings.Split(s, "\n")

	for _, line := range lines {
		line := strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, protectedKeyHRP) {
			encrypted, err := bech32ishUnpack(line)
			if err != nil {
				return nil, err
			}

			//passphrase, err := readPassphrase()
			passphrase := "asdf"
			if err != nil {
				return nil, fmt.Errorf("failed to read passphrase")
			}

			identity, err := age.NewScryptIdentity(string(passphrase))
			if err != nil {
				return nil, err
			}

			reader := bytes.NewReader(encrypted)
			decryptor, err := age.Decrypt(reader, identity)
			if err != nil {
				return nil, err
			}

			var decrypted bytes.Buffer
			_, err = io.Copy(&decrypted, decryptor)
			if err != nil {
				return nil, err
			}

			secretKey := secretKeyHRP + decrypted.String()

			identities, err := age.ParseIdentities(strings.NewReader(secretKey))
			if err != nil {
				return nil, err
			}

			if len(identities) == 0 {
				return nil, fmt.Errorf("invalid secret key")
			}

			return identities[0], nil
		}
	}

	return nil, fmt.Errorf("invalid protected key")
}

func exportGeneratedKey(key *age.X25519Identity, targetFile string, passphrase []byte) error {
	var target io.WriteCloser
	target, err := os.Create(targetFile)
	if err != nil {
		return err
	}

	public := key.Recipient().String()
	private := key.String()

	if len(passphrase) == 0 {
		fmt.Fprintln(os.Stderr, "no passphrase given, generated key will be stored in clear text")
	} else {
		passPhraseRecipient, err := age.NewScryptRecipient(string(passphrase))
		if err != nil {
			return err
		}

		var buf bytes.Buffer
		encrypter, err := age.Encrypt(&buf, passPhraseRecipient)
		if err != nil {
			return err
		}

		// The secret key is bech32 encoded, we just trim off the HRP
		private = strings.TrimPrefix(private, secretKeyHRP)
		privateData := strings.NewReader(private)

		io.Copy(encrypter, privateData)
		encrypter.Close()

		private, err = bech32ishPack(protectedKeyHRP, buf.Bytes())
	}

	timestamp := time.Now().Format(time.RFC3339)
	_, err = fmt.Fprintf(target, "# created: %v\n# public key: %v\n%v\n", timestamp, public, private)
	if err != nil {
		return err
	}

	return target.Close()
}

const bech32Charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

var bech32Encoding = base32.NewEncoding(bech32Charset).WithPadding(base32.NoPadding)

func bech32ishUnpack(s string) ([]byte, error) {
	splitPoint := strings.LastIndex(s, "1")
	if splitPoint < 1 || splitPoint+7 > len(s) {
		return nil, fmt.Errorf("unexpected hrp data")
	}

	encoded := s[splitPoint+1:]
	encoded = strings.ToLower(encoded)
	decoder := base32.NewDecoder(bech32Encoding, strings.NewReader(encoded))

	var decoded bytes.Buffer
	_, err := io.Copy(&decoded, decoder)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack: %w", err)
	}

	data := decoded.Bytes()
	return data, nil
}

func bech32ishPack(hrp string, data []byte) (string, error) {
	var encoded bytes.Buffer
	encoder := base32.NewEncoder(bech32Encoding, &encoded)

	_, err := io.Copy(encoder, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	encoder.Close()

	upper := strings.ToUpper(encoded.String())
	return hrp + "1" + upper, nil
}
