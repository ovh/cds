package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/vault/meta"
	"github.com/ryanuber/columnize"
)

// MountsCommand is a Command that lists the mounts.
type MountsCommand struct {
	meta.Meta
}

func (c *MountsCommand) Run(args []string) int {
	flags := c.Meta.FlagSet("mounts", meta.FlagSetDefault)
	flags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := flags.Parse(args); err != nil {
		return 1
	}

	client, err := c.Client()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error initializing client: %s", err))
		return 2
	}

	mounts, err := client.Sys().ListMounts()
	if err != nil {
		c.Ui.Error(fmt.Sprintf(
			"Error reading mounts: %s", err))
		return 2
	}

	paths := make([]string, 0, len(mounts))
	for path := range mounts {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	columns := []string{"Path | Type | Default TTL | Max TTL | Description"}
	for _, path := range paths {
		mount := mounts[path]
		defTTL := "system"
		switch {
		case mount.Type == "system":
			defTTL = "n/a"
		case mount.Type == "cubbyhole":
			defTTL = "n/a"
		case mount.Config.DefaultLeaseTTL != 0:
			defTTL = strconv.Itoa(mount.Config.DefaultLeaseTTL)
		}
		maxTTL := "system"
		switch {
		case mount.Type == "system":
			maxTTL = "n/a"
		case mount.Type == "cubbyhole":
			maxTTL = "n/a"
		case mount.Config.MaxLeaseTTL != 0:
			maxTTL = strconv.Itoa(mount.Config.MaxLeaseTTL)
		}
		columns = append(columns, fmt.Sprintf(
			"%s | %s | %s | %s | %s", path, mount.Type, defTTL, maxTTL, mount.Description))
	}

	c.Ui.Output(columnize.SimpleFormat(columns))
	return 0
}

func (c *MountsCommand) Synopsis() string {
	return "Lists mounted backends in Vault"
}

func (c *MountsCommand) Help() string {
	helpText := `
Usage: vault mounts [options]

  Outputs information about the mounted backends.

  This command lists the mounted backends, their mount points, the
  configured TTLs, and a human-friendly description of the mount point.
  A TTL of 'system' indicates that the system default is being used.

General Options:
` + meta.GeneralOptionsUsage()
	return strings.TrimSpace(helpText)
}
