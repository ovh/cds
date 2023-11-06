package main

import (
	"os"
)

func main() {
	cmd := cmdMain()
	cmd.AddCommand(cmdVersion)

	if isCI() {
		if isLegacyMode() {
			cmd.AddCommand(cmdExport)
			cmd.AddCommand(cmdUpload())
			cmd.AddCommand(cmdArtifacts())
			cmd.AddCommand(cmdDownload())
			cmd.AddCommand(cmdTmpl())
			cmd.AddCommand(cmdCheckSecret())
			cmd.AddCommand(cmdTag())
			cmd.AddCommand(cmdRun())
			cmd.AddCommand(cmdExit())
			cmd.AddCommand(cmdRegister())
			cmd.AddCommand(cmdCache())
			cmd.AddCommand(cmdKey())
			cmd.AddCommand(cmdJunitParser())
			cmd.AddCommand(cmdCDSVersionSet())
			cmd.AddCommand(cmdRunResult())
		} else {
			cmd.AddCommand(CmdResult())
		}
	} else {
		// last command: doc, this command is hidden
		cmd.AddCommand(cmdDoc(cmd))
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
