package template

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/ovh/cds/engine/venom"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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

	ts := venom.TestSuite{
		Name: "Title of TestSuite",
		TestCases: []venom.TestCase{
			{
				Name: "TestCase with default value, exec cmd. Check if exit code != 1",
				TestSteps: []venom.TestStep{
					{
						"type":   "exec",
						"script": "echo 'foo'",
					},
				},
			},
			{
				Name: "Title of First TestCase",
				TestSteps: []venom.TestStep{
					{
						"script":     "echo 'foo'",
						"assertions": []string{"Result.Code ShouldEqual 0"},
					},
					{
						"script":     "echo 'bar'",
						"assertions": []string{"Result.StdOut ShouldNotContainSubstring bar"},
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
