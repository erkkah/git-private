package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
)

func Exists(path AbsolutePath) (bool, error) {
	_, err := os.Stat(path.Absolute())
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func GetFileHash(path RepoRelativePath) (string, error) {
	absolute, err := RepoAbsolute(path)
	if err != nil {
		return "", err
	}
	file, err := absolute.Open()
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
