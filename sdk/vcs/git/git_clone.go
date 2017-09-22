package git

import (
	"fmt"
	"strings"
	"time"
)

// CloneOpts is a optional structs for git clone command
type CloneOpts struct {
	Depth                   int
	SingleBranch            bool
	Branch                  string
	Recursive               bool
	Verbose                 bool
	Quiet                   bool
	CheckoutCommit          string
	NoStrictHostKeyChecking bool
}

// Clone make a git clone
func Clone(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) error {
	if verbose {
		t1 := time.Now()
		if opts != nil && opts.CheckoutCommit != "" {
			defer LogFunc("Checkout commit %s", opts.CheckoutCommit)
		}
		defer LogFunc("Git clone %s (%v s)", path, int(time.Since(t1).Seconds()))
	}

	var commands []cmd
	repoURL, err := getRepoURL(repo, auth)
	if err != nil {
		return err
	}

	commands = prepareGitCloneCommands(repoURL, path, opts)
	return runGitCommands(repo, commands, auth, output)
}

func prepareGitCloneCommands(repo string, path string, opts *CloneOpts) cmds {
	allCmd := []cmd{}
	gitcmd := cmd{
		cmd:  "git",
		args: []string{"clone"},
	}

	if opts != nil {
		if opts.Quiet {
			gitcmd.args = append(gitcmd.args, "--quiet")
		} else if opts.Verbose {
			gitcmd.args = append(gitcmd.args, "--verbose")
		}

		if opts.CheckoutCommit == "" {
			if opts.Depth != 0 {
				gitcmd.args = append(gitcmd.args, "--depth", fmt.Sprintf("%d", opts.Depth))
			}
		}

		if opts.Branch != "" {
			gitcmd.args = append(gitcmd.args, "--branch", opts.Branch)
		} else if opts.SingleBranch {
			gitcmd.args = append(gitcmd.args, "--single-branch")
		}

		if opts.Recursive {
			gitcmd.args = append(gitcmd.args, "--recursive")
		}
	}

	gitcmd.args = append(gitcmd.args, repo)

	if path != "" {
		gitcmd.args = append(gitcmd.args, path)
	}

	allCmd = append(allCmd, gitcmd)

	if opts != nil && opts.CheckoutCommit != "" {
		resetCmd := cmd{
			cmd:  "git",
			args: []string{"reset", "--hard", opts.CheckoutCommit},
		}
		//Locate the git reset cmd to the right directory
		if path == "" {
			t := strings.Split(repo, "/")
			resetCmd.dir = strings.TrimSuffix(t[len(t)-1], ".git")
		} else {
			resetCmd.dir = path
		}

		allCmd = append(allCmd, resetCmd)
	}

	return cmds(allCmd)
}
