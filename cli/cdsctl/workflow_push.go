package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
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
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	VariadicArgs: cli.Arg{
		Name: "yaml-file",
	},
	Flags: []cli.Flag{
		{
			Kind:  reflect.Bool,
			Name:  "skip-update-files",
			Usage: "Useful if you don't want to update yaml files after pushing the workflow.",
		},
	},
}

func workflowPushRun(c cli.Values) error {
	// Get the file
	files := strings.Split(c.GetString("yaml-file"), ",")

	// Create a buffer to write our archive to.
	buf := new(bytes.Buffer)
	var dir string

	// Create a new tar archive.
	filesToRead := []string{}
	for _, file := range files {
		fi, err := os.Lstat(file)
		if err != nil {
			fmt.Printf("Skipping file %s: %v\n", file, err)
			continue
		}

		//Skip the directory
		if fi.IsDir() {
			continue
		}

		fmt.Println("Reading file ", cli.Magenta(file))
		if dir == "" {
			dir = filepath.Dir(file)
		}
		if dir != filepath.Dir(file) {
			return fmt.Errorf("files must be ine the same directory")
		}

		filesToRead = append(filesToRead, file)
	}

	if len(filesToRead) == 0 {
		return fmt.Errorf("wrong usage: you should specify your workflow YAML files. See %s workflow push --help for more details", os.Args[0])
	}

	if err := workflowFilesToTarWriter(filesToRead, buf); err != nil {
		return err
	}

	// Open the tar archive for reading.
	btes := buf.Bytes()
	r := bytes.NewBuffer(btes)

	// Push it !
	msgList, tr, err := client.WorkflowPush(c.GetString(_ProjectKey), r)
	for _, msg := range msgList {
		fmt.Println(msg)
	}

	if err != nil {
		return err
	}

	fmt.Println("Workflow successfully pushed !")

	if c.GetBool("skip-update-files") {
		return nil
	}

	return workflowTarReaderToFiles(dir, tr, false, false)
}

func workflowFilesToTarWriter(files []string, buf io.Writer) error {
	tw := tar.NewWriter(buf)

	// Add some files to the archive.
	for _, file := range files {
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
	return tw.Close()
}
