package main

import (
	"github.com/spf13/cobra"
)

func cmdMain(w *currentWorker) *cobra.Command {
	var mainCmd = &cobra.Command{
		Use:   "worker",
		Short: "CDS Worker",
		Long: `A pipeline is structured in sequential stages containing
one or multiple concurrent jobs. A Job will be executed by a worker.

The worker provides some useful commands that can be used in a step, as ` + "`worker upload...`" + `, ` + "`worker download...`" + "`worker cache...`" + `

On Windows OS, theses commands can be accessed with ` + "`worker.exe [cmd]` syntax.",
		Run: runCmd(w),
	}

	initFlagsRun(mainCmd)

	return mainCmd
}
