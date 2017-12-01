package git

import (
	"fmt"
)

// TagOpts represents options for git tag command
type TagOpts struct {
	Message  string
	SignKey  string
	SignID   string
	Name     string
	Path     string
	Username string
}

// TagCreate makes git tag action
func TagCreate(repo string, auth *AuthOpts, opts *TagOpts, output *OutputOpts) error {
	var commands []cmd
	repoURL, err := getRepoURL(repo, auth)
	if err != nil {
		return err
	}
	commands = prepareGitTagCreateCommands(repoURL, opts)
	return runGitCommands(repo, commands, auth, output)
}

// TagList List tag from given git directory
func TagList(repo, dir string, auth *AuthOpts, output *OutputOpts) error {
	repoURL, err := getRepoURL(repo, auth)
	if err != nil {
		return err
	}
	commands := prepareGitTagListCommands(repoURL, dir)
	return runGitCommands(repo, commands, auth, output)
}

func prepareGitTagCreateCommands(repo string, opts *TagOpts) cmds {
	allCmd := []cmd{}

	if opts != nil && opts.SignKey != "" {
		// Create command to import key
		importpubCmd := cmd{
			cmd:  "gpg",
			args: []string{"--import", "pgp.pub.key"},
		}
		allCmd = append(allCmd, importpubCmd)

		importcmd := cmd{
			cmd:  "gpg",
			args: []string{"--import", "pgp.key"},
		}

		allCmd = append(allCmd, importcmd)
	}

	allCmd = append(allCmd, gitConfigCommand("user.name", opts.Username))
	allCmd = append(allCmd, gitConfigCommand("user.email", "cds@localhost"))

	gitcmd := cmd{
		cmd:  "git",
		args: []string{"tag"},
	}

	// Option for git push after tagging
	optPush := &PushOpts{}

	if opts != nil {
		if opts.Path != "" {
			gitcmd.dir = opts.Path
			optPush.Directory = opts.Path
		}
		gitcmd.args = append(gitcmd.args, "-a", opts.Name)

		if opts.Message != "" {
			gitcmd.args = append(gitcmd.args, "-m", fmt.Sprintf("\"%s\"", opts.Message))
		}
		if opts.SignKey != "" {
			gitcmd.args = append(gitcmd.args, "-u", opts.SignID)
		}
		optPush.Branch = opts.Name
	}

	allCmd = append(allCmd, gitcmd)
	allCmd = append(allCmd, prepareGitPushCommands(repo, optPush)...)
	return cmds(allCmd)
}

func prepareGitTagListCommands(repo, dir string) cmds {
	allCmd := []cmd{}

	gitcmd := cmd{
		cmd:  "git",
		args: []string{"ls-remote", "--tags", "--refs", repo},
		dir:  dir,
	}

	allCmd = append(allCmd, gitcmd)
	return cmds(allCmd)
}
