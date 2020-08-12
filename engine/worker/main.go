package main

func main() {
	cmd := cmdMain()
	cmd.AddCommand(cmdExport)
	cmd.AddCommand(cmdUpload())
	cmd.AddCommand(cmdArtifacts())
	cmd.AddCommand(cmdDownload())
	cmd.AddCommand(cmdTmpl())
	cmd.AddCommand(cmdCheckSecret())
	cmd.AddCommand(cmdTag())
	cmd.AddCommand(cmdRun())
	cmd.AddCommand(cmdExit())
	cmd.AddCommand(cmdVersion)
	cmd.AddCommand(cmdRegister())
	cmd.AddCommand(cmdCache())
	cmd.AddCommand(cmdKey())
	cmd.AddCommand(cmdJunitParser())
	cmd.AddCommand(cmdCDSVersionSet())

	// last command: doc, this command is hidden
	cmd.AddCommand(cmdDoc(cmd))

	cmd.Execute()
}
