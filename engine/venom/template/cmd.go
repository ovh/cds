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
				Name: "TestCase with default value, check if code < 300 on a GET URL",
				TestSteps: []sdk.TestStep{
					{
						URL: "http://youapp.localhost/status",
					},
				},
			},
			{
				Name: "TestCase with default value, exec cmd. Check if exit code != 1",
				TestSteps: []sdk.TestStep{
					{
						Command: "cds status",
					},
				},
			},
			{
				Name: "Title of First TestCase",
				TestSteps: []sdk.TestStep{
					{
						Type:       "exec",
						Command:    "cds",
						Args:       []string{"status"},
						Assertions: []string{"code ShouldEqual 0"},
					},
					{
						Type:       "exec",
						Command:    "cds",
						Args:       []string{"user", "list"},
						StdIn:      "Content stdin",
						Assertions: []string{"code ShouldNotEqual 0"},
					},
				},
			},
			{
				Name: "Title of Second TestCase",
				TestSteps: []sdk.TestStep{
					{
						Type:       "http",
						Method:     "GET",
						URL:        "http://youapp.localhost/status",
						Assertions: []string{"code ShouldBeBetweenOrEqual 200 299"},
					},
					{
						Type:    "http",
						Method:  "POST",
						URL:     "http://youapp.localhost/status",
						Payload: "{\"foo\":\"bar\"}",
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
