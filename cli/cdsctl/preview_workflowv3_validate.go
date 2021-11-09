package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk/workflowv3"
)

var workflowV3ValidateCmd = cli.Command{
	Name:  "workflowv3-validate",
	Short: "Parse and validate given Workflow V3 files.",
	Ctx: []cli.Arg{
		{Name: _ProjectKey},
	},
	VariadicArgs: cli.Arg{
		Name: "yaml-file",
	},
	Flags: []cli.Flag{
		{
			Name: "silent",
			Type: cli.FlagBool,
		},
	},
}

func workflowV3ValidateRun(v cli.Values) error {
	projectKey := v.GetString(_ProjectKey)

	if _, err := client.ProjectGet(v.GetString(_ProjectKey)); err != nil {
		return errors.WithMessage(err, "cannot get project")
	}

	var files []string
	filesPath := strings.Split(v.GetString("yaml-file"), ",")
	for _, p := range filesPath {
		if err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return errors.Wrapf(err, "cannot read given path")
		}
	}

	workflowIn := workflowv3.NewWorkflow()
	for i := range files {
		buf, err := os.ReadFile(files[i])
		if err != nil {
			return errors.Wrapf(err, "cannot read file at %q", files[i])
		}
		var w workflowv3.Workflow
		if err := yaml.Unmarshal(buf, &w); err != nil {
			return errors.Wrapf(err, "cannot unmarshal file %q", files[i])
		}
		if err := workflowIn.Add(w); err != nil {
			return errors.WithMessagef(err, "cannot merge workflow content from file %q", files[i])
		}
	}

	silent := v.GetBool("silent")
	if !silent {
		fmt.Printf("Workflow read from %d file(s) %q:\n", len(files), files)
		buf, err := yaml.Marshal(workflowIn)
		if err != nil {
			return err
		}
		fmt.Println(string(buf))
	}

	if err := workflowV3Validate(projectKey, workflowIn, silent); err != nil {
		return err
	}

	return nil
}

func workflowV3Validate(projectKey string, workflowIn workflowv3.Workflow, silent bool) error {
	// Static validation for workflow
	extDep, err := workflowIn.Validate()
	if err != nil {
		return err
	}

	if !silent {
		fmt.Println("Workflow is valid.")
		buf, err := json.MarshalIndent(extDep, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("Detected external deps:\n%s\n", buf)
	}

	return nil
}
