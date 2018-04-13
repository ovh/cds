package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ovh/cds/cli"
)

var workflowPullCmd = cli.Command{
	Name:  "pull",
	Short: "Pull a workflow",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
		{Name: _WorkflowName},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.String,
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
		},
		{
			Kind:    reflect.Bool,
			Name:    "with-permissions",
			Usage:   "Export permissions",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "force",
			Usage:   "Force, may override files",
			Default: "false",
		},
		{
			Kind:    reflect.Bool,
			Name:    "quiet",
			Usage:   "If true, do not output filename created",
			Default: "false",
		},
	},
}

func workflowPullRun(c cli.Values) error {
	dir := strings.TrimSpace(c.GetString("output-dir"))
	if dir == "" {
		dir = "."
	}
	if err := os.MkdirAll(dir, os.FileMode(0744)); err != nil {
		return fmt.Errorf("Unable to create directory %s: %v", c.GetString("output-dir"), err)
	}

	tr, err := client.WorkflowPull(c.GetString(_ProjectKey), c.GetString(_WorkflowName), c.GetBool("with-permissions"))
	if err != nil {
		return err
	}

	return workflowTarReaderToFiles(dir, tr, c.GetBool("force"), c.GetBool("quiet"))
}

func workflowTarReaderToFiles(dir string, tr *tar.Reader, force, quiet bool) error {
	if tr == nil {
		return fmt.Errorf("Unable to read workflow")
	}
	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error while reading the tar archive err:%v", err)
		}

		fname := filepath.Join(dir, hdr.Name)
		if _, err = os.Stat(fname); err == nil || os.IsExist(err) {
			if !force {
				if !cli.AskForConfirmation(fmt.Sprintf("This will override %s. Do you want to continue?", fname)) {
					os.Exit(0)
				}
			}
		}

		if verbose {
			fmt.Println("Creating file", cli.Magenta(fname))
		}
		fi, err := os.Create(fname)
		if err != nil {
			return fmt.Errorf("Error while creating file %s err:%v", fname, err)
		}
		if _, err := io.Copy(fi, tr); err != nil {
			return fmt.Errorf("Error while writing file %s err:%v", fname, err)
		}
		if err := fi.Close(); err != nil {
			return fmt.Errorf("Error while closing file %s err:%v", fname, err)
		}
		if !quiet {
			fmt.Println(fname)
		}
	}
	return nil
}
