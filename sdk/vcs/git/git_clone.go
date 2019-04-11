package git

import (
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
)

// CloneOpts is a optional structs for git clone command
type CloneOpts struct {
	Depth                   int
	SingleBranch            bool
	Branch                  string
	Tag                     string
	Recursive               bool
	Verbose                 bool
	Quiet                   bool
	CheckoutCommit          string
	NoStrictHostKeyChecking bool
}

// Clone make a git clone
func Clone(repo string, path string, auth *AuthOpts, opts *CloneOpts, output *OutputOpts) (string, error) {
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
		return "", err
	}

	var userLogCommand string
	userLogCommand, commands = prepareGitCloneCommands(repoURL, path, opts)
	return userLogCommand, runGitCommands(repo, commands, auth, output)
}

func prepareGitCloneCommands(repo string, path string, opts *CloneOpts) (string, cmds) {
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

		if opts.Depth != 0 {
			gitcmd.args = append(gitcmd.args, "--depth", fmt.Sprintf("%d", opts.Depth))
		}

		if opts.Branch != "" || (opts.Tag != "" && opts.Tag != sdk.DefaultGitCloneParameterTagValue) {
			if opts.Tag != "" && opts.Tag != sdk.DefaultGitCloneParameterTagValue {
				gitcmd.args = append(gitcmd.args, "--branch", opts.Tag)
			} else {
				gitcmd.args = append(gitcmd.args, "--branch", opts.Branch)
			}
		} else if opts.SingleBranch {
			gitcmd.args = append(gitcmd.args, "--single-branch")
		}

		if opts.Recursive {
			gitcmd.args = append(gitcmd.args, "--recursive")
		}
	}

	userLogCommand := "Executing: git " + strings.Join(gitcmd.args, " ") + " ...  "
	gitcmd.args = append(gitcmd.args, repo)

	if path != "" {
		gitcmd.args = append(gitcmd.args, path)
	}

	allCmd = append(allCmd, gitcmd)

	// if a specific commit hash is given, try to reset current repo to this commit
	if opts != nil && opts.CheckoutCommit != "" && opts.Tag == "" {
		// if no branch or tag given, this means that we cloned the repo on the default branch, we need to fetch the target commit hash
		// fetching a specific commit hash will not work for old git version (1.7 for example)
		if opts.Branch == "" && opts.Tag == "" {
			fetchCmd := cmd{
				cmd:  "git",
				args: []string{"fetch", "origin", opts.CheckoutCommit},
			}
			userLogCommand += "\n\rExecuting: git " + strings.Join(fetchCmd.args, " ")
			//Locate the git reset cmd to the right directory
			if path == "" {
				t := strings.Split(repo, "/")
				fetchCmd.dir = strings.TrimSuffix(t[len(t)-1], ".git")
			} else {
				fetchCmd.dir = path
			}

			allCmd = append(allCmd, fetchCmd)
		}

		// even if we cloned the right branch or tag, we need to reset to the given commit hash that could be different than HEAD
		resetCmd := cmd{
			cmd:  "git",
			args: []string{"reset", "--hard", opts.CheckoutCommit},
		}
		userLogCommand += "\n\rExecuting: git " + strings.Join(resetCmd.args, " ")
		//Locate the git reset cmd to the right directory
		if path == "" {
			t := strings.Split(repo, "/")
			resetCmd.dir = strings.TrimSuffix(t[len(t)-1], ".git")
		} else {
			resetCmd.dir = path
		}

		allCmd = append(allCmd, resetCmd)
	}

	return userLogCommand, cmds(allCmd)
}
