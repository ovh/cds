package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

func TestShellListCommand(t *testing.T) {
	prjs := []sdk.Project{{Key: "prj-1"}, {Key: "prj-2"}}
	wkfs := []sdk.Workflow{{Name: "wkf-1"}, {Name: "wkf-2"}}

	// important, if not set there could be os.Exit in test that will stop testing before end
	cli.ShellMode = true

	c := shellCurrent{
		tree: cli.NewCommand(cli.Command{Name: "cdsctl"}, nil, []*cobra.Command{
			cli.NewCommand(cli.Command{Name: "version"}, func(v cli.Values) error { return nil }, nil),
			cli.NewCommand(cli.Command{Name: "project"}, nil, []*cobra.Command{
				cli.NewListCommand(cli.Command{Name: "list"}, func(v cli.Values) (cli.ListResult, error) {
					return cli.AsListResult(prjs), nil
				}, nil, cli.CommandWithExtraFlags, cli.CommandWithExtraAliases),
				cli.NewCommand(cli.Command{
					Name: "show",
					Ctx:  []cli.Arg{{Name: _ProjectKey}},
				}, func(v cli.Values) error { return nil }, nil),
				cli.NewCommand(cli.Command{Name: "create"}, func(v cli.Values) error { return nil }, nil),
				cli.NewCommand(cli.Command{
					Name: "export",
					Ctx:  []cli.Arg{{Name: _ProjectKey}},
				}, func(v cli.Values) error { return nil }, nil),
				cli.NewCommand(cli.Command{Name: "workflow"}, nil, []*cobra.Command{
					cli.NewListCommand(cli.Command{
						Name: "list",
						Ctx:  []cli.Arg{{Name: _ProjectKey}},
					}, func(v cli.Values) (cli.ListResult, error) {
						return cli.AsListResult(wkfs), nil
					}, nil, cli.CommandWithExtraFlags, cli.CommandWithExtraAliases),
					cli.NewCommand(cli.Command{
						Name: "show",
						Ctx:  []cli.Arg{{Name: _ProjectKey}, {Name: _WorkflowName}},
					}, func(v cli.Values) error { return nil }, nil),
					cli.NewCommand(cli.Command{
						Name: "create",
						Ctx:  []cli.Arg{{Name: _ProjectKey}},
					}, func(v cli.Values) error { return nil }, nil),
					cli.NewCommand(cli.Command{
						Name: "export",
						Ctx:  []cli.Arg{{Name: _ProjectKey}, {Name: _WorkflowName}},
					}, func(v cli.Values) error { return nil }, nil),
				}),
				cli.NewCommand(cli.Command{Name: "group"}, nil, []*cobra.Command{
					cli.NewCommand(cli.Command{
						Name: "import",
						Ctx:  []cli.Arg{{Name: _ProjectKey}},
					}, func(v cli.Values) error { return nil }, nil),
				}),
			}),
		}),
	}

	tests := []struct {
		path     string
		out      []string
		subMenus []string
		cmds     []string
	}{{
		path:     "/",
		out:      nil,
		subMenus: []string{"project"},
		cmds:     []string{"version"},
	}, {
		path:     "/project",
		out:      []string{"prj-1", "prj-2"},
		subMenus: nil,
		cmds:     []string{"create"},
	}, {
		path:     "/project/prj-1",
		out:      nil,
		subMenus: []string{"group", "workflow"},
		cmds:     []string{"export"},
	}, {
		path:     "/project/prj-1/group",
		out:      nil,
		subMenus: nil,
		cmds:     []string{"import"},
	}, {
		path:     "/project/prj-1/workflow",
		out:      []string{"wkf-1", "wkf-2"},
		subMenus: nil,
		cmds:     []string{"create"},
	}, {
		path:     "/project/prj-1/workflow/wkf-1",
		out:      nil,
		subMenus: nil,
		cmds:     []string{"export"},
	}}

	for _, test := range tests {
		out, subMenus, cmds, _ := c.shellListCommand(test.path, nil, false)
		assert.Equal(t, test.out, out, fmt.Sprintf("check out for path %s", test.path))
		assert.Equal(t, test.subMenus, subMenus, fmt.Sprintf("check sub menus for path %s", test.path))
		assert.Equal(t, test.cmds, cmds, fmt.Sprintf("check commands for path %s", test.path))
	}
}
