package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

const PrivateExtension = ".private"
const ToolName = "git-private"

const PrivateKeyVariable = "GIT_PRIVATE_KEY"
const PrivateKeyFileVariable = "GIT_PRIVATE_KEYFILE"

func privateDir() string {
	if val, exists := os.LookupEnv("GIT_PRIVATE_DIR"); exists {
		return val
	}
	return ".gitprivate"
}

func StateDir() (AbsolutePath, error) {
	dir := privateDir()
	if filepath.IsAbs(dir) {
		return AbsolutePath(dir), nil
	}
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}
	path := root.Join(RepoRelativePath(privateDir()))
	return path, nil
}

func KeysFile() (AbsolutePath, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return dir.Join("keys.dat"), nil
}

func PathsFile() (AbsolutePath, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return dir.Join("paths.json"), nil
}

func EnsureInitialized() error {
	dir, err := StateDir()
	if err != nil {
		return err
	}
	exists, err := Exists(dir)
	if exists {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("not initialized, run '%s init'", ToolName)
}
