package git

import (
	"bytes"
	"fmt"
	"strings"
)

// Info contains some Information about a git repository
type Info struct {
	GitDescribe string
	Hash        string // git Hash
	Message     string
	Author      string
	AuthorEmail string
	Branch      string
}

// ExtractInfo returns an info, containing git information (git.Hash, describe)
func ExtractInfo(dir string) (Info, error) {
	info := Info{}
	cmdHash := []cmd{{dir: dir, cmd: "git", args: []string{"rev-parse", "HEAD"}}}
	cmdDescribe := []cmd{{dir: dir, cmd: "git", args: []string{"describe", "--tags"}}}
	cmdMessage := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%s", "-1"}}}
	cmdAuthor := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%an", "-1"}}}
	cmdAuthorEmail := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%ae", "-1"}}}
	cmdCurrentBranch := []cmd{{dir: dir, cmd: "git", args: []string{"rev-parse", "--abbrev-ref", "HEAD"}}}

	var err error
	if info.Hash, err = gitRawCommandString(cmdHash); err != nil {
		return info, err
	}
	if info.GitDescribe, err = gitRawCommandString(cmdDescribe); err != nil {
		return info, err
	}
	if info.Message, err = gitRawCommandString(cmdMessage); err != nil {
		return info, err
	}
	if info.Author, err = gitRawCommandString(cmdAuthor); err != nil {
		return info, err
	}
	if info.AuthorEmail, err = gitRawCommandString(cmdAuthorEmail); err != nil {
		return info, err
	}
	if info.Branch, err = gitRawCommandString(cmdCurrentBranch); err != nil {
		return info, err
	}
	return info, nil
}

func gitRawCommandString(c cmds) (string, error) {
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	if err := runGitCommandRaw(c, output); err != nil {
		return "", fmt.Errorf("Error while running git command: %s", err)
	}

	if len(stdErr.Bytes()) > 0 {
		return "", fmt.Errorf("Error while running git command (stdErr): %s", stdErr.String())
	}

	if len(stdOut.Bytes()) > 0 {
		// search for version
		lines := strings.Split(stdOut.String(), "\n")
		if len(lines) == 0 {
			return "", fmt.Errorf("Error while getting information, more than one line: %s", stdOut.Bytes())
		}
		return lines[0], nil
	}
	return "", fmt.Errorf("Error while getting information (empty returns)")
}
