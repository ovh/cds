package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var adminRepositoriesCmd = cli.Command{
	Name:  "repositories",
	Short: "Manage CDS repositories uService",
}

func adminRepositories() *cobra.Command {
	return cli.NewCommand(adminRepositoriesCmd, nil, []*cobra.Command{
		cli.NewListCommand(adminRepositorisStatusCmd, adminRepositorisStatusRun, nil),
		cli.NewListCommand(adminRepositoriesListCmd, adminRepositoriesListRun, nil),
	})
}

func adminRepositorisStatusRun(_ cli.Values) (cli.ListResult, error) {
	services, err := client.ServicesByType(sdk.TypeRepositories)
	if err != nil {
		return nil, err
	}
	status := sdk.MonitoringStatus{}
	for _, srv := range services {
		status.Lines = append(status.Lines, srv.MonitoringStatus.Lines...)
	}
	return cli.AsListResult(status.Lines), nil
}

var adminRepositorisStatusCmd = cli.Command{
	Name:    "status",
	Short:   "display the status of repositories",
	Example: "cdsctl admin repositories status",
}

var adminRepositoriesListCmd = cli.Command{
	Name:  "list",
	Short: "list the git repositories stored on disk by each repositories service instance, with their size",
	Example: `cdsctl admin repositories list
cdsctl admin repositories list --name repositories-01
cdsctl admin repositories list --filter expired=true --format json`,
	Flags: []cli.Flag{
		{
			Name:    "name",
			Usage:   "only query the repositories service instance with this name",
			Default: "",
		},
	},
}

// adminRepositoryLine is one git repository of one instance, as displayed.
type adminRepositoryLine struct {
	Instance       string `cli:"instance"`
	URL            string `cli:"url,key"`
	Size           string `cli:"size"`
	Expired        bool   `cli:"expired"`
	ProtectedUntil string `cli:"protected_until"`
	ID             string // directory name (base64 of the URL), shown with --verbose
}

// adminRepositoriesLines flattens per-instance listings into display lines,
// largest repositories first, then by instance and URL.
func adminRepositoriesLines(lists []sdk.RepositoriesAdminList) []adminRepositoryLine {
	type sized struct {
		line adminRepositoryLine
		size int64
	}
	var all []sized
	for _, l := range lists {
		for _, r := range l.Repositories {
			line := adminRepositoryLine{
				Instance: l.Instance,
				URL:      r.URL,
				Size:     humanize.IBytes(uint64(r.Size)),
				Expired:  r.Expired,
				ID:       r.ID,
			}
			if l.ComputedAt == nil {
				line.Size = "unknown"
			}
			if r.ProtectedUntil != nil {
				line.ProtectedUntil = r.ProtectedUntil.Local().Format(time.RFC3339)
			}
			all = append(all, sized{line: line, size: r.Size})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.size != b.size {
			return a.size > b.size
		}
		if a.line.Instance != b.line.Instance {
			return a.line.Instance < b.line.Instance
		}
		return a.line.URL < b.line.URL
	})
	lines := make([]adminRepositoryLine, 0, len(all))
	for _, s := range all {
		lines = append(lines, s.line)
	}
	return lines
}

func adminRepositoriesListRun(v cli.Values) (cli.ListResult, error) {
	var names []string
	if name := v.GetString("name"); name != "" {
		names = []string{name}
	} else {
		srvs, err := client.ServicesByType(sdk.TypeRepositories)
		if err != nil {
			return nil, err
		}
		for _, srv := range srvs {
			names = append(names, srv.Name)
		}
	}

	var lists []sdk.RepositoriesAdminList
	for _, name := range names {
		bts, err := client.ServiceNameCallGET(name, "/admin/repositories")
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to query repositories service %q: %v\n", name, err)
			continue
		}
		var list sdk.RepositoriesAdminList
		if err := sdk.JSONUnmarshal(bts, &list); err != nil {
			fmt.Fprintf(os.Stderr, "invalid answer from repositories service %q: %v\n", name, err)
			continue
		}
		lists = append(lists, list)
	}
	return cli.AsListResult(adminRepositoriesLines(lists)), nil
}
