package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/cli"
)

var workflowPushCmd = cli.Command{
	Name:  "push",
	Short: "Push a workflow",
	Long: `
		Useful when you want to push a workflow and his dependencies (pipelines, applications, environments)
		For example if you have a workflow with pipelines build and tests you can push your workflow and pipelines with
		cdsctl workflow push tests.pip.yml build.pip.yml myWorkflow.yml
	`,
	Args: []cli.Arg{
		{Name: "project-key"},
	},
	VariadicArgs: cli.Arg{
		Name: "yaml-file",
	},
}

func workflowPushRun(c cli.Values) error {
	// Get the file
	files := strings.Split(c.GetString("yaml-file"), ",")

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)

	// Create a new tar archive.
	tw := tar.NewWriter(buf)

	// Add some files to the archive.
	for _, file := range files {
		fmt.Println("Reading file ", file)
		filBuf, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		hdr := &tar.Header{
			Name: filepath.Base(file),
			Mode: 0600,
			Size: int64(len(filBuf)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if n, err := tw.Write(filBuf); err != nil {
			return err
		} else if n == 0 {
			return fmt.Errorf("nothing to write")
		}
	}
	// Make sure to check the error on Close.
	if err := tw.Close(); err != nil {
		return err
	}

	// Open the tar archive for reading.
	btes := buf.Bytes()
	r := bytes.NewBuffer(btes)

	// Push it !
	msgList, err := client.WorkflowPush(c.GetString("project-key"), r)
	for _, msg := range msgList {
		fmt.Println(msg)
	}

	return err
}
