package utils

import (
	"os"
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

func GetGitRootPath() (AbsolutePath, error) {
	root, code, err := runGitCommand("rev-parse", "--path-format=relative", "--show-toplevel")
	if code != 0 {
		return "", err
	}

	root = strings.TrimSpace(root)
	root = path.Clean(root)
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}

	return AbsolutePath(root), nil
}

func getIgnoreFilePath() (AbsolutePath, error) {
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}

	ignoreFile := root.Join(".gitignore")
	return ignoreFile, nil
}

func readIgnoreFile() ([]string, error) {
	ignoreFile, err := getIgnoreFilePath()
	if err != nil {
		return nil, err
	}

	exists, err := Exists(ignoreFile)
	if err != nil {
		return []string{}, err
	}
	if !exists {
		return []string{}, nil
	}

	contents, err := os.ReadFile(ignoreFile.Absolute())
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
	err = os.WriteFile(ignoreFile.Absolute(), []byte(strings.Join(lines, "\n")), 0660)
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

type AbsolutePath string
type RepoRelativePath string

func (ap AbsolutePath) Join(relative RepoRelativePath) AbsolutePath {
	return AbsolutePath(path.Join(string(ap), string(relative)))
}

func (ap AbsolutePath) Open() (*os.File, error) {
	return os.Open(ap.Absolute())
}

func (ap AbsolutePath) Absolute() string {
	return string(ap)
}

func (rp RepoRelativePath) Relative() string {
	return string(rp)
}

// RepoRelative converts the given path to a path relative to the current repo.
func RepoRelative(path AbsolutePath) (RepoRelativePath, error) {
	var err error

	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}

	relative, err := filepath.Rel(root.Absolute(), path.Absolute())
	if err != nil {
		return "", err
	}

	return RepoRelativePath(relative), nil
}

// RepoAbsolute converts the given path to an absolute path.
// Provided paths must be relative to the current repo.
func RepoAbsolute(pathWithinRepo RepoRelativePath) (AbsolutePath, error) {
	root, err := GetGitRootPath()
	if err != nil {
		return "", err
	}

	return root.Join(pathWithinRepo), nil
}
