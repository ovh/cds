package git

// PushOpts represents options for git push command
type PushOpts struct {
	Directory string
	Remote    string
	Branch    string
}

// Push make git push action
func Push(repo string, auth *AuthOpts, opts *PushOpts, output *OutputOpts) error {
	var commands []cmd
	repoURL, err := getRepoURL(repo, auth)
	if err != nil {
		return err
	}
	commands = prepareGitPushCommands(repoURL, opts)
	return runGitCommands(repo, commands, auth, output)
}

func prepareGitPushCommands(repoURL string, opts *PushOpts) cmds {
	allCmd := []cmd{}
	gitcmd := cmd{
		cmd:  "git",
		args: []string{"push"},
	}

	if opts != nil {
		if opts.Directory != "" {
			gitcmd.dir = opts.Directory
		}
		gitcmd.args = append(gitcmd.args, repoURL, opts.Branch)
	}
	allCmd = append(allCmd, gitcmd)
	return cmds(allCmd)
}
