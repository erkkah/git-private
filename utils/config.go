package utils

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

const SecretsExtension = ".secret"
const ToolName = "git-private"

func privateDir() string {
	if val, exists := os.LookupEnv("GIT_PRIVATE_DIR"); exists {
		return val
	}
	return ".gitprivate"
}

func StateDir() (string, error) {
	dir := privateDir()
	if filepath.IsAbs(dir) {
		return dir, nil
	}
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}
	path := path.Join(root, privateDir())
	return path, nil
}

func KeysFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "keys.json"), nil
}

func PathsFile() (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return path.Join(dir, "paths.json"), nil
}

func EnsureInitialized() error {
	dir, err := StateDir()
	if err != nil {
		return err
	}
	if Exists(dir) {
		return nil
	}
	return fmt.Errorf("not initialized, run '%s init'", ToolName)
}
