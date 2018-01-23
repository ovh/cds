package repo

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type Repo struct {
	path string
}

func New(path string) (Repo, error) {
	return Repo{path}, nil
}

func (r Repo) FetchURL() (string, error) {
	cmd := exec.Command("git", "remote", "show", "origin", "-n")
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	cmd.Dir = r.path
	cmd.Stderr = stdErr
	cmd.Stdout = stdOut
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	errOut := stdErr.Bytes()
	if len(errOut) > 0 {
		return "", fmt.Errorf(string(errOut))
	}

	reader := bufio.NewReader(stdOut)
	var fetchURL string
	for {
		b, _, err := reader.ReadLine()
		if err == io.EOF || b == nil {
			break
		}
		if err != nil {
			return "", err
		}
		s := string(b)
		if strings.Contains(s, "Fetch URL:") {
			fetchURL = strings.Replace(s, "Fetch URL:", "", 1)
			fetchURL = strings.TrimSpace(fetchURL)
		}
	}

	return fetchURL, nil
}

func (r Repo) Name() (string, error) {
	fetchURL, err := r.FetchURL()
	if err != nil {
		return "", err
	}

	return trimURL(fetchURL)
}

func (r Repo) LocalConfigGet(section, key string) (string, error) {
	cmd := exec.Command("git", "config", "--local", "--get", fmt.Sprintf("%s.%s", section, key))
	stdOut := new(bytes.Buffer)
	stdErr := new(bytes.Buffer)
	cmd.Dir = r.path
	cmd.Stderr = stdErr
	cmd.Stdout = stdOut
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	errOut := stdErr.Bytes()
	if len(errOut) > 0 {
		return "", fmt.Errorf(string(errOut))
	}

	s := stdOut.String()
	return s[:len(s)-1], nil
}

func (r Repo) LocalConfigSet(section, key, value string) error {
	conf, _ := r.LocalConfigGet(section, key)
	s := fmt.Sprintf("%s.%s", section, key)
	if conf != "" {
		cmd := exec.Command("git", "config", "--local", "--unset", s)
		cmd.Dir = r.path
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	cmd := exec.Command("git", "config", "--local", "--add", s, value)
	stdErr := new(bytes.Buffer)
	cmd.Dir = r.path
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return err
	}

	errOut := stdErr.Bytes()
	if len(errOut) > 0 {
		return fmt.Errorf(string(errOut))
	}

	return nil
}

func trimURL(fetchURL string) (string, error) {
	repoName := fetchURL

	if strings.HasSuffix(repoName, ".git") {
		repoName = repoName[:len(repoName)-4]
	}

	if strings.HasPrefix(repoName, "https://") {
		repoName = repoName[8:]
		for strings.Count(repoName, "/") > 1 {
			firstSlash := strings.Index(repoName, "/")
			if firstSlash == -1 {
				return "", fmt.Errorf("invalid url")
			}
			repoName = repoName[firstSlash+1:]
		}
		return repoName, nil
	}

	if strings.HasPrefix(repoName, "ssh://") {
		// ssh://[user@]server/project.git
		repoName = repoName[6:]
		firstSlash := strings.Index(repoName, "/")
		if firstSlash == -1 {
			return "", fmt.Errorf("invalid url")
		}
		repoName = repoName[firstSlash+1:]
	} else {
		// [user@]server:project.git
		firstSemicolon := strings.Index(repoName, ":")
		if firstSemicolon == -1 {
			return "", fmt.Errorf("invalid url")
		}
		repoName = repoName[firstSemicolon+1:]
	}

	return repoName, nil
}
