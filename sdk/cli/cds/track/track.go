package track

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/ovh/cds/sdk"
)

var display string

func print() {
	clear := "\r"
	w, _, _ := terminal.GetSize(1)
	for i := 0; i < w; i++ {
		clear += " "
	}

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			if display == "" {
				continue
			}

			fmt.Printf(clear + "\r" + display)
			display = ""
		}
	}()
}

func track(hash string) {
	display = fmt.Sprintf("Looking for %s...", hash)
	print()

	// Look for Pipeline build
	var pbs []sdk.PipelineBuild
	var err error
	for i := 0; i < 10; i++ {
		pbs, err = sdk.GetBuildingPipelineByHash(hash)
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		sdk.Exit("\nError: Cannot find any pipeline build (%s)\n", err)
	}

	//fmt.Printf("Found %d pipeline builds\n", len(pbs))
	var pbI int
	for pbI < len(pbs) {
		pb := pbs[pbI]

		// Update pipeline status and display
		for {
			upb, err := sdk.GetPipelineBuildStatus(pb.Pipeline.ProjectKey,
				pb.Application.Name, pb.Pipeline.Name, pb.Environment.Name, pb.BuildNumber)
			if err == nil {
				pb = upb
				formatDisplay(pb)
			}

			time.Sleep(500 * time.Millisecond)

			upbs, err := sdk.GetBuildingPipelineByHash(hash)
			if err == nil {
				pbs = upbs
			}

			if pb.Status != sdk.StatusBuilding {
				fmt.Printf("\n")
				//fmt.Printf(" <- %s Done !\n", pb.Pipeline.Name)
				pbI++
				break
			}
		}
	}

	// Pipeline finished, display result long enough
	time.Sleep(1 * time.Second)
	fmt.Printf("\n")
	os.Exit(0)
}

func formatDisplay(pb sdk.PipelineBuild) {
	//yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintfFunc()
	blue := color.New(color.FgBlue).SprintfFunc()
	magenta := color.New(color.FgMagenta).SprintfFunc()
	green := color.New(color.FgGreen).SprintfFunc()
	cyan := color.New(color.FgCyan).SprintfFunc()

	buildingChar := blue("↻")
	okChar := green("✓")
	koChar := red("✗")
	arrow := cyan("➤")

	projKey := pb.Pipeline.ProjectKey
	appName := pb.Application.Name
	pipeline := pb.Pipeline.Name
	branch := pb.Trigger.VCSChangesBranch
	env := pb.Environment.Name
	version := pb.Version
	status := pb.Status

	switch status {
	case sdk.StatusBuilding:
		display = buildingChar
	case sdk.StatusSuccess:
		display = okChar
	case sdk.StatusFail:
		display = koChar
	}
	display = fmt.Sprintf("%s %s%s%s%s%s%s", display, projKey, arrow, appName, arrow, pipeline, cyan("#%d", version))
	if branch != "" {
		display = fmt.Sprintf("%s (%s)", display, magenta(branch))
	}
	if env != "NoEnv" {
		display += fmt.Sprintf(" %s", magenta(env))
	}

	// Format actions
	for _, s := range pb.Stages {
		for _, ab := range s.ActionBuilds {
			display += " " + formatActionBuild(ab)
		}
	}
}

func formatActionBuild(ab sdk.ActionBuild) string {
	yellow := color.New(color.FgYellow).SprintfFunc()
	red := color.New(color.FgRed).SprintfFunc()
	blue := color.New(color.FgBlue).SprintfFunc()
	green := color.New(color.FgGreen).SprintfFunc()

	switch ab.Status {
	case sdk.StatusSuccess:
		return green("[%s]", ab.ActionName)
	case sdk.StatusFail:
		return red("[%s]", ab.ActionName)
	case sdk.StatusBuilding:
		return blue("[%s]", ab.ActionName)
	case sdk.StatusWaiting:
		return yellow("[%s]", ab.ActionName)
	default:
		return ""
	}

}
