package trigger

import (
	"errors"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/spf13/cobra"
)

// Cmd for pipeline operation
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "List, add, clone, get or remove triggers between pipelines",
		Long:  ``,
	}

	cmd.AddCommand(addTriggerCmd())
	cmd.AddCommand(showTriggerCmd())
	cmd.AddCommand(listTriggerCmd())
	cmd.AddCommand(deleteTriggerCmd())
	cmd.AddCommand(copyTriggerCmd())

	return cmd
}

func triggerFromString(src, dst string) (*sdk.PipelineTrigger, error) {
	t := &sdk.PipelineTrigger{}

	srcT := strings.Split(src, "/")
	if len(srcT) < 3 || len(srcT) > 4 {
		return nil, errors.New("wrong source pipeline format, should be <project>/<application>/<pipeline>[/<env>]")
	}
	dstT := strings.Split(dst, "/")
	if len(dstT) < 3 || len(dstT) > 4 {
		return nil, errors.New("wrong destination pipeline format, should be <project>/<application>/<pipeline>[/<env>]")
	}

	t.SrcProject.Key = srcT[0]
	t.SrcApplication.Name = srcT[1]
	t.SrcPipeline.Name = srcT[2]
	if len(srcT) == 4 {
		t.SrcEnvironment.Name = srcT[3]
	} else {
		t.SrcEnvironment.Name = sdk.DefaultEnv.Name
	}

	t.DestProject.Key = dstT[0]
	t.DestApplication.Name = dstT[1]
	t.DestPipeline.Name = dstT[2]
	if len(dstT) == 4 {
		t.DestEnvironment.Name = dstT[3]
	} else {
		t.DestEnvironment.Name = sdk.DefaultEnv.Name
	}

	return t, nil
}

func triggersEqual(lhs, rhs *sdk.PipelineTrigger) bool {
	if lhs.SrcEnvironment.Name == "" {
		lhs.SrcEnvironment.Name = sdk.DefaultEnv.Name
	}
	if lhs.DestEnvironment.Name == "" {
		lhs.DestEnvironment.Name = sdk.DefaultEnv.Name
	}
	if rhs.SrcEnvironment.Name == "" {
		rhs.SrcEnvironment.Name = sdk.DefaultEnv.Name
	}
	if rhs.DestEnvironment.Name == "" {
		rhs.DestEnvironment.Name = sdk.DefaultEnv.Name
	}

	return lhs.DestProject.Key == rhs.DestProject.Key &&
		lhs.DestApplication.Name == rhs.DestApplication.Name &&
		lhs.DestPipeline.Name == rhs.DestPipeline.Name &&
		lhs.DestEnvironment.Name == rhs.DestEnvironment.Name
}
