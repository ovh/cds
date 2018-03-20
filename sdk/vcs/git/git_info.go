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
// ignore error if a command fails (example: for empty repository)
func ExtractInfo(dir string) Info {
	info := Info{}
	cmdHash := []cmd{{dir: dir, cmd: "git", args: []string{"rev-parse", "HEAD"}}}
	cmdDescribe := []cmd{{dir: dir, cmd: "git", args: []string{"describe", "--tags"}}}
	cmdMessage := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%B", "-1"}}}
	cmdAuthor := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%an", "-1"}}}
	cmdAuthorEmail := []cmd{{dir: dir, cmd: "git", args: []string{"log", "--format=%ae", "-1"}}}
	cmdCurrentBranch := []cmd{{dir: dir, cmd: "git", args: []string{"rev-parse", "--abbrev-ref", "HEAD"}}}

	// git rev-parse HEAD can fail with
	// "fatal: ambiguous argument 'HEAD': unknown revision or path not in the working tree."
	// ignore err
	info.Hash, _ = gitRawCommandString(cmdHash)

	// git log --format=... can fail with
	// "fatal: your current branch 'master' does not have any commits yet"
	// ignore err
	info.Message, _ = gitRawCommandString(cmdMessage)

	info.Author, _ = gitRawCommandString(cmdAuthor)
	info.AuthorEmail, _ = gitRawCommandString(cmdAuthorEmail)
	info.Branch, _ = gitRawCommandString(cmdCurrentBranch)

	// git describe can fail with
	// "fatal: No names found, cannot describe anything."
	// ignore err
	info.GitDescribe, _ = gitRawCommandString(cmdDescribe)
	return info
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
