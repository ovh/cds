package git

func gitConfigCommand(key string, value string) cmd {
	gitcmd := cmd{
		cmd:  "git",
		args: []string{"config", "--global", key, value},
	}
	return gitcmd
}
