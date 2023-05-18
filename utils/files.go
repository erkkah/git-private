package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func Touch(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func GetFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	sum := hash.Sum([]byte{})
	encodedLen := hex.EncodedLen(len(sum))
	encoded := make([]byte, encodedLen)
	hex.Encode(encoded, sum)
	return string(encoded), nil
}

func ReadFromFileOrStdin(file string) (string, error) {
	var data []byte
	var err error
	if file == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return "", err
	}

	key := string(data)
	key = strings.TrimSpace(key)

	return key, nil
}
