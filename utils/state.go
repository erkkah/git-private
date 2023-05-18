package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/agessh"
)

type KeyType string

const (
	SSH KeyType = "ssh"
	AGE KeyType = "age"
)

type Key struct {
	Type     KeyType
	Key      string
	ID       string
	ReadOnly bool
}

type KeyList struct {
	Version int
	Keys    []Key
}

type SecureFile struct {
	Path string
	Hash string
}

type FileList struct {
	Version int
	Files   []SecureFile
}

func LoadKeyList(identity age.Identity) (KeyList, error) {
	var list KeyList
	file, err := KeysFile()
	if err != nil {
		return KeyList{}, err
	}

	exists, err := Exists(file)
	if err != nil {
		return KeyList{}, err
	}

	if !exists {
		return KeyList{}, nil
	}

	reader, err := os.Open(file)
	if err != nil {
		return KeyList{}, err
	}
	defer reader.Close()

	decrypted, err := age.Decrypt(reader, identity)
	if err != nil {
		return KeyList{}, fmt.Errorf("key list decryption failed")
	}

	err = loadFrom(decrypted, &list)
	return list, err
}

func StoreKeyList(identity age.Identity, list KeyList) error {
	// Make sure the current user has access to the key list before replacing it
	_, err := GetRecipients(identity)
	if err != nil {
		return err
	}

	// Get recipients with read/write access
	recipients, err := getRecipientsFromKeylist(list, ReadWrite)
	if err != nil {
		return err
	}

	if len(recipients) == 0 {
		return fmt.Errorf("cannot update key list, no keys with read/write access")
	}

	file, err := KeysFile()
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	encrypted, err := age.Encrypt(&buf, recipients...)
	if err != nil {
		return err
	}

	list.Version = 1
	err = storeTo(encrypted, &list)
	if err != nil {
		return err
	}

	err = encrypted.Close()
	if err != nil {
		return err
	}

	err = os.WriteFile(file, buf.Bytes(), 0600)
	if err != nil {
		return err
	}

	return nil
}

func LoadFileList() (FileList, error) {
	file, err := PathsFile()
	if err != nil {
		return FileList{}, err
	}
	var list FileList
	err = load(file, &list)
	return list, err
}

func StoreFileList(list FileList) error {
	file, err := PathsFile()
	if err != nil {
		return err
	}
	list.Version = 1
	return store(file, &list)
}

func loadFrom(reader io.Reader, dest interface{}) error {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, dest)
	if err != nil {
		return err
	}
	return nil
}

func load(file string, dest interface{}) error {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, dest)
	if err != nil {
		return err
	}
	return nil
}

func storeTo(writer io.Writer, src interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(data)
	_, err = io.Copy(writer, reader)

	if err != nil {
		return err
	}
	return nil
}

func store(file string, src interface{}) error {
	bytes, err := json.Marshal(src)
	if err != nil {
		return err
	}
	err = os.WriteFile(file, bytes, 0600)
	if err != nil {
		return err
	}
	return nil
}

type KeyAccess string

const (
	ReadOnly  KeyAccess = "ro"
	ReadWrite KeyAccess = "rw"
)

func getRecipientsFromKeylist(keyList KeyList, access KeyAccess) ([]age.Recipient, error) {
	var recipients []age.Recipient
	var err error

	for _, key := range keyList.Keys {
		if access == ReadWrite && key.ReadOnly {
			continue
		}
		var recipient age.Recipient

		if key.Type == AGE {
			parsedRecipients, err := age.ParseRecipients(strings.NewReader(key.Key))
			if err != nil {
				return nil, err
			}
			if len(parsedRecipients) != 1 {
				return nil, fmt.Errorf("unexpected key contents")
			}
			recipient = parsedRecipients[0]
		} else if key.Type == SSH {
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

func GetRecipients(identity age.Identity) ([]age.Recipient, error) {
	keyList, err := LoadKeyList(identity)
	if err != nil {
		return nil, err
	}

	return getRecipientsFromKeylist(keyList, ReadOnly)
}
