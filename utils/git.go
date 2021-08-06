package utils

import (
	"io/ioutil"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

func runGitCommand(args ...string) (string, int, error) {
	cmd := exec.Command("git", args...)
	bytes, err := cmd.Output()
	exitCode := cmd.ProcessState.ExitCode()
	if exitCode != -1 {
		err = nil
	}
	return string(bytes), exitCode, err
}

func IsInsideGitTree() (bool, error) {
	_, code, err := runGitCommand("rev-parse", "--is-inside-work-tree")
	if code == 0 {
		return true, nil
	}
	return false, err
}

func GetGitRootPath() (string, error) {
	root, code, err := runGitCommand("rev-parse", "--show-toplevel")
	if code == 0 {
		return path.Clean(strings.TrimSpace(root)), nil
	}
	return "", err
}

func getIgnoreFilePath() (string, error) {
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}

	ignoreFile := path.Join(root, ".gitignore")
	return ignoreFile, nil
}

func readIgnoreFile() ([]string, error) {
	ignoreFile, err := getIgnoreFilePath()
	if err != nil {
		return nil, err
	}

	if !Exists(ignoreFile) {
		return []string{}, nil
	}

	contents, err := ioutil.ReadFile(ignoreFile)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(contents), "\n")
	return lines, nil
}

func writeIgnoreFile(lines []string) error {
	ignoreFile, err := getIgnoreFilePath()
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(ignoreFile, []byte(strings.Join(lines, "\n")), 0660)
	if err != nil {
		return err
	}
	return nil
}

func GitRemoveIgnorePattern(pattern string) error {
	lines, err := readIgnoreFile()
	if err != nil {
		return err
	}

	var updatedLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != strings.TrimSpace(pattern) {
			updatedLines = append(updatedLines, line)
		}
	}

	err = writeIgnoreFile(updatedLines)
	if err != nil {
		return err
	}
	return nil
}

func GitAddIgnorePattern(pattern string) error {
	lines, err := readIgnoreFile()
	if err != nil {
		return err
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == pattern {
			return nil
		}
	}

	lines = append(lines, pattern)
	err = writeIgnoreFile(lines)
	if err != nil {
		return err
	}

	return nil
}

func IsGitIgnored(fileName string) (bool, error) {
	_, code, err := runGitCommand("check-ignore", "-q", fileName)
	if code == 0 {
		return true, nil
	}
	return false, err
}

// RepoRelative converts the given path to a path relative to the current repo.
func RepoRelative(path string) (string, error) {
	var err error

	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}

	path, err = filepath.Rel(root, path)
	if err != nil {
		return "", err
	}

	return path, nil
}
