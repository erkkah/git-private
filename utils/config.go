package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

func secretsDir() string {
	if val, exists := os.LookupEnv("SECRETS_DIR"); exists {
		return val
	}
	return ".gitsecrets"
}

func SecretsDir() (string, error) {
	dir := secretsDir()
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}
	path := path.Join(root, secretsDir())
	return path, nil
}

func KeysFile() (string, error) {
	dir, err := SecretsDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "keys.json"), nil
}

func PathsFile() (string, error) {
	dir, err := SecretsDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "paths.json"), nil
}

const SecretsExtension = ".secret"

func EnsureInitialized() error {
	dir, err := SecretsDir()
	if err != nil {
		return err
	}
	if Exists(dir) {
		return nil
	}
	return fmt.Errorf("not initialized, run 'git-secrets init'")
}
