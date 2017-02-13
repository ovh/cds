package template

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
)

// Cmd run
var Cmd = &cobra.Command{
	Use:   "template",
	Short: "Export a TestSuite Template",
	Run: func(cmd *cobra.Command, args []string) {
		template()
	},
}

func template() {

	ts := sdk.TestSuite{
		Name: "Title of TestSuite",
		TestCases: []sdk.TestCase{
			{
				Name: "TestCase with default value, exec cmd. Check if exit code != 1",
				TestSteps: []sdk.TestStep{
					{
						ScriptContent: "cds status",
					},
				},
			},
			{
				Name: "Title of First TestCase",
				TestSteps: []sdk.TestStep{
					{
						ScriptContent: "cds status",
						Assertions:    []string{"code ShouldEqual 0"},
					},
					{
						ScriptContent: "cds user list",
						Assertions:    []string{"code ShouldNotEqual 0"},
					},
				},
			},
		},
	}

	out, err := yaml.Marshal(ts)
	if err != nil {
		log.Fatalf("Err:%s", err)
	}

	fmt.Printf("%s\n", string(out))

}
