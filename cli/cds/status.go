package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/gosuri/uilive"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
)

var (
	stream           bool
	execMsg, execErr []string
)

func init() {
	statusCmd.Flags().BoolVarP(&stream, "stream", "s", false, "stream status --stream.")
	statusCmd.Flags().StringSliceVarP(&execMsg, "exec", "", nil, `Exec a cmd on each status KO: --stream --exec 'myLights --pulse red --duration=1000'`)
	statusCmd.Flags().StringSliceVarP(&execErr, "execErr", "", nil, `Exec a cmd on each error while requesting cds: --stream --exec 'myLights --pulse blue --duration=1000' --execErr 'myLights --pulse red --duration=2000'`)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Retrieve CDS api status",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if !stream {
			out := status(nil, cmd, args)
			for _, s := range out.Lines {
				fmt.Println(s.String())
			}
			return
		}

		writer := uilive.New()
		writer.Start()
		for {
			out := status(writer, cmd, args)
			for _, s := range out.Lines {
				fmt.Fprintf(writer, s.String()+"\n")
			}

			if err := writer.Flush(); err != nil {
				fmt.Printf("Error while flushing: %s", err)
			}
			processStatusLine(out.Lines)
			processWait()
		}
	},
}

func status(w *uilive.Writer, cmd *cobra.Command, args []string) *sdk.MonitoringStatus {
	output, err := sdk.GetStatus()
	if err != nil {
		if !stream {
			sdk.Exit("Cannot get status (%s)\n", err)
		}
		processExecError(err)
	}
	return output
}

func processWait() {
	// see https://github.com/briandowns/spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	time.Sleep(5 * time.Second)
	s.Stop()
}

func processExecError(err error) {
	fmt.Printf("Error:%s", err)
	for _, ex := range execErr {
		execCmd(ex)
	}
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Start()
	time.Sleep(5 * time.Second)
	s.Stop()
}

func processStatusLine(lines []sdk.MonitoringStatusLine) {
	for _, l := range lines {
		if stream && l.Status == sdk.MonitoringStatusOK {
			continue
		}
		for _, ex := range execMsg {
			execCmd(ex)
		}
	}
}

func execCmd(toExec string) {
	opts := strings.Split(toExec, " ")
	if toExec != "" {
		if _, err := exec.LookPath(opts[0]); err != nil {
			sdk.Exit("Invalid --exec path for %s, err: %s", opts[0], err.Error())
			return
		}

		s := spinner.New(spinner.CharSets[35], 100*time.Millisecond)
		cmd := exec.Command(opts[0], opts[1:]...)
		s.Start()
		if err := cmd.Start(); err != nil {
			fmt.Printf("Error: %s", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			fmt.Printf("Error: %s\n", err)
		}
		s.Stop()
	}
}
