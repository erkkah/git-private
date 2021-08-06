package utils

import (
	"encoding/json"
	"io/ioutil"
)

type KeyType string

const (
	SSH KeyType = "ssh"
	AGE KeyType = "age"
)

type Key struct {
	Type KeyType
	Key  string
	ID   string
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

func LoadKeyList() (KeyList, error) {
	var list KeyList
	file, err := KeysFile()
	if err != nil {
		return KeyList{}, err
	}
	err = load(file, &list)
	return list, err
}

func StoreKeyList(list KeyList) error {
	file, err := KeysFile()
	if err != nil {
		return err
	}
	list.Version = 1
	return store(file, &list)
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

func load(file string, dest interface{}) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, dest)
	if err != nil {
		return err
	}
	return nil
}

func store(file string, dest interface{}) error {
	bytes, err := json.Marshal(dest)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(file, bytes, 0600)
	if err != nil {
		return err
	}
	return nil
}
