package main

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

func testInitCurrentShell() shellCurrent {
	prjs := []sdk.Project{{Key: "prj-1"}, {Key: "prj-2"}}
	wkfs := []sdk.Workflow{{Name: "wkf-1"}, {Name: "wkf-2"}}
	app := struct {
		Name string `cli:"name,key"`
	}{Name: "app-1"}

	return shellCurrent{
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
						Args: []cli.Arg{{Name: _ApplicationName}},
					}, func(v cli.Values) error { return nil }, nil),
					cli.NewCommand(cli.Command{
						Name: "export",
						Ctx:  []cli.Arg{{Name: _ProjectKey}, {Name: _WorkflowName}},
					}, func(v cli.Values) error { return nil }, nil),
				}),
				cli.NewCommand(cli.Command{Name: "application"}, nil, []*cobra.Command{
					cli.NewGetCommand(cli.Command{
						Name: "show",
						Ctx:  []cli.Arg{{Name: _ProjectKey}, {Name: _ApplicationName}},
					}, func(v cli.Values) (interface{}, error) { return app, nil }, nil),
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
}

func TestShellListCommand(t *testing.T) {
	// important, if not set there could be os.Exit in test that will stop testing before end
	cli.ShellMode = true

	c := testInitCurrentShell()

	tests := []struct {
		path     string
		out      string
		items    []string
		subMenus []string
		cmds     []string
	}{{
		path:     "/",
		out:      "",
		items:    nil,
		subMenus: []string{"project"},
		cmds:     []string{"version"},
	}, {
		path:     "/project",
		out:      "prj-1\nprj-2\n",
		items:    []string{"prj-1", "prj-2"},
		subMenus: nil,
		cmds:     []string{"create"},
	}, {
		path:     "/project/prj-1",
		out:      "",
		items:    nil,
		subMenus: []string{"group", "workflow"},
		cmds:     []string{"export"},
	}, {
		path:     "/project/prj-1/group",
		out:      "",
		items:    nil,
		subMenus: nil,
		cmds:     []string{"import"},
	}, {
		path:     "/project/prj-1/workflow",
		out:      "wkf-1\nwkf-2\n",
		items:    []string{"wkf-1", "wkf-2"},
		subMenus: nil,
		cmds:     []string{"create"},
	}, {
		path:     "/project/prj-1/workflow/wkf-1",
		out:      "",
		items:    nil,
		subMenus: nil,
		cmds:     []string{"export"},
	}, {
		path:     "/project/prj-1/application/app-1",
		out:      "name      app-1\n",
		items:    nil,
		subMenus: nil,
		cmds:     nil,
	}}

	for _, test := range tests {
		out, items, subMenus, cmds, _ := c.shellListCommand(test.path, nil, false)
		assert.Equal(t, test.out, out, fmt.Sprintf("check out value for path %s", test.path))
		assert.Equal(t, test.items, items, fmt.Sprintf("check items for path %s", test.path))
		assert.Equal(t, test.subMenus, subMenus, fmt.Sprintf("check sub menus for path %s", test.path))
		assert.Equal(t, test.cmds, cmds, fmt.Sprintf("check commands for path %s", test.path))
	}
}

func TestFindCmd(t *testing.T) {
	// important, if not set there could be os.Exit in test that will stop testing before end
	cli.ShellMode = true

	tests := []struct {
		path, expected, home string
		notFound             bool
	}{
		{"/", "/", "", false},
		{"/project", "/project", "", false},
		{"/project/prj-1", "/project/prj-1", "", false},
		{"/project/prj-1/workflow", "/project/prj-1/workflow", "", false},
		{"/project/prj-1/workflow/wkf-1", "/project/prj-1/workflow/wkf-1", "", false},
		{"/project/prj-1/variable", "", "", true},
		{"/project/unknown", "", "", true},
		{"/../../././//../project", "/project", "", false},
		{"", "/", "/", false},
		{"", "/project", "/project", false},
	}

	for _, test := range tests {
		c := testInitCurrentShell()
		c.home = test.home
		found := c.cdCmd([]string{test.path})
		assert.Equal(t, !test.notFound, found, fmt.Sprintf("check that found is %t", !test.notFound))
		assert.Equal(t, test.expected, c.path, fmt.Sprintf("check result for path '%s'", test.path))
	}
}
