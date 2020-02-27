package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	survey "gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var templateBulkCmd = cli.Command{
	Name:    "bulk",
	Short:   "Bulk apply CDS workflow template and push all given workflows",
	Example: "cdsctl template bulk group-name/template-slug -i PROJ1/workflow1 -i PROJ1/workflow2 -p PROJ1/workflow1:repo=github.com/ovh/cds",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Type:      cli.FlagArray,
			Name:      "instances",
			ShortHand: "i",
			Usage:     "Specify instances path",
			Default:   "",
		},
		{
			Type:      cli.FlagArray,
			Name:      "params",
			ShortHand: "p",
			Usage:     "Specify parameters for template like --params PROJ1/workflow1:paramKey=paramValue",
			Default:   "",
		},
		{
			Type:    cli.FlagArray,
			Name:    "detach",
			Usage:   "Set to generate a workflow detached from the template like --detach PROJ1/workflow1",
			Default: "",
		},
		{
			Name:  "instances-file",
			Usage: "Specify path|url of a json|yaml file that contains instances with params",
		},
		{
			Type:  cli.FlagBool,
			Name:  "track",
			Usage: "Wait the bulk to be over",
		},
	},
}

type templateBulkInstancePath struct {
	ProjectKey, WorkflowName string
}

func (t templateBulkInstancePath) Key() string {
	return fmt.Sprintf("%s/%s", t.ProjectKey, t.WorkflowName)
}

type templateBulkParameter struct {
	InstancePath templateBulkInstancePath
	Key, Value   string
}

type templateBulkFile struct {
	TemplatePath string                     `json:"template_path" yaml:"template_path"`
	Instances    []templateBulkFileInstance `json:"instances" yaml:"instances"`
}

type templateBulkFileInstance struct {
	WorkflowPath string   `json:"workflow_path" yaml:"workflow_path"`
	Parameters   []string `json:"parameters" yaml:"parameters"`
}

func templateExtractAndValidateInstances(instanceKeys []string) (map[string]templateBulkInstancePath, error) {
	minstances := make(map[string]templateBulkInstancePath)
	for i := range instanceKeys {
		// instance path should be formatted like MYPROJ/myWorkflow
		instancePath := strings.Split(instanceKeys[i], "/")
		if len(instancePath) != 2 {
			return nil, fmt.Errorf("invalid given instance path %s", instanceKeys[i])
		}

		minstances[instanceKeys[i]] = templateBulkInstancePath{
			ProjectKey:   instancePath[0],
			WorkflowName: instancePath[1],
		}
	}

	return minstances, nil
}

func templateExtractAndValidateParams(rawParams []string) ([]templateBulkParameter, error) {
	var params []templateBulkParameter
	for i := range rawParams {
		err := fmt.Errorf("invalid given parameter %s", rawParams[i])

		// instance path should be formatted like MYPROJ/myWorkflow:myParameterKey=myValue
		param := strings.Split(rawParams[i], "=")
		if len(param) != 2 {
			return nil, err
		}
		paramKey := strings.Split(param[0], ":")
		if len(paramKey) != 2 {
			return nil, err
		}
		instancePath := strings.Split(paramKey[0], "/")
		if len(paramKey) != 2 {
			return nil, err
		}

		params = append(params, templateBulkParameter{
			InstancePath: templateBulkInstancePath{
				ProjectKey:   instancePath[0],
				WorkflowName: instancePath[1],
			},
			Key:   paramKey[1],
			Value: param[1],
		})
	}

	return params, nil
}

func templateExtractAndValidateFileParams(filePath string) (*sdk.WorkflowTemplate, []sdk.WorkflowTemplateBulkOperation, error) {
	if filePath == "" {
		return nil, nil, nil
	}

	contentFile, format, err := exportentities.OpenPath(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer contentFile.Close() //nolint

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(contentFile); err != nil {
		return nil, nil, fmt.Errorf("cannot read from given file")
	}

	var f templateBulkFile
	var errU error
	switch format {
	case exportentities.FormatJSON:
		errU = json.Unmarshal(buf.Bytes(), &f)
	case exportentities.FormatYAML:
		errU = yaml.Unmarshal(buf.Bytes(), &f)
	default:
		return nil, nil, exportentities.ErrUnsupportedFormat
	}
	if errU != nil {
		return nil, nil, fmt.Errorf("cannot unmarshal given file %v", err)
	}

	groupName, templateSlug, err := cli.ParsePath(f.TemplatePath)
	if err != nil {
		return nil, nil, err
	}

	// try to get the template from cds
	template, err := client.TemplateGet(groupName, templateSlug)
	if err != nil {
		return nil, nil, err
	}

	operations := make([]sdk.WorkflowTemplateBulkOperation, len(f.Instances))
	for i, instance := range f.Instances {
		// instance path should be formatted like MYPROJ/myWorkflow
		instancePath := strings.Split(instance.WorkflowPath, "/")
		if len(instancePath) != 2 {
			return nil, nil, fmt.Errorf("invalid given instance path %s", instance.WorkflowPath)
		}

		operation := sdk.WorkflowTemplateBulkOperation{
			Request: sdk.WorkflowTemplateRequest{
				ProjectKey:   instancePath[0],
				WorkflowName: instancePath[1],
				Parameters:   make(map[string]string),
			},
		}

		for _, value := range instance.Parameters {
			// instance path should be formatted like myParameterKey=myValue
			param := strings.Split(value, "=")
			if len(param) != 2 {
				return nil, nil, fmt.Errorf("invalid given parameter value")
			}
			operation.Request.Parameters[param[0]] = param[1]
		}

		operations[i] = operation
	}

	return template, operations, nil
}

func templateInitOperationFromParams(mwtis map[string]sdk.WorkflowTemplateInstance, fileOperations []sdk.WorkflowTemplateBulkOperation,
	minstances map[string]templateBulkInstancePath, params []templateBulkParameter) map[string]sdk.WorkflowTemplateBulkOperation {
	// for all given instances, create an operation and reuse request if instance already exists
	moperations := make(map[string]sdk.WorkflowTemplateBulkOperation, len(minstances))
	for key, i := range minstances {
		if instance, ok := mwtis[key]; ok {
			moperations[key] = sdk.WorkflowTemplateBulkOperation{
				Request: instance.Request,
			}
		} else {
			moperations[key] = sdk.WorkflowTemplateBulkOperation{
				Request: sdk.WorkflowTemplateRequest{
					ProjectKey:   i.ProjectKey,
					WorkflowName: i.WorkflowName,
					Parameters:   make(map[string]string),
				},
			}
		}
	}

	// for all given file params, create or enrich existing operation but do not use existing instance
	for _, operation := range fileOperations {
		key := fmt.Sprintf("%s/%s", operation.Request.ProjectKey, operation.Request.WorkflowName)
		moperations[key] = operation
	}

	// for all given params, create an operation and reuse request if instance already exists
	for _, param := range params {
		key := param.InstancePath.Key()
		if _, ok := moperations[key]; !ok {
			if instance, ok := mwtis[key]; ok {
				moperations[key] = sdk.WorkflowTemplateBulkOperation{
					Request: instance.Request,
				}
			} else {
				moperations[key] = sdk.WorkflowTemplateBulkOperation{
					Request: sdk.WorkflowTemplateRequest{
						ProjectKey:   param.InstancePath.ProjectKey,
						WorkflowName: param.InstancePath.WorkflowName,
						Parameters:   make(map[string]string),
					},
				}
			}
		}
	}

	// populate operations with params values,
	for _, p := range params {
		key := p.InstancePath.Key()
		if moperations[key].Request.Parameters == nil {
			o := moperations[key]
			o.Request.Parameters = make(map[string]string)
			moperations[key] = o
		}
		moperations[key].Request.Parameters[p.Key] = p.Value
	}

	return moperations
}

func templateAskForInstances(wt *sdk.WorkflowTemplate, mwtis map[string]sdk.WorkflowTemplateInstance, minstances map[string]templateBulkInstancePath,
	moperations map[string]sdk.WorkflowTemplateBulkOperation) error {
	opts := make([]cli.CustomMultiSelectOption, len(mwtis))
	values := make(map[string]sdk.WorkflowTemplateInstance, len(mwtis))
	i := 0
	for key, instance := range mwtis {
		notUpToDate := instance.WorkflowTemplateVersion < wt.Version

		var info string
		if instance.Workflow == nil {
			info = cli.Yellow("not imported")
		} else if notUpToDate {
			info = cli.Red("not up to date")
		} else {
			info = cli.Green("up to date")
		}

		_, instanceGivenAsParam := moperations[instance.Key()]
		// selected by default if given as param or if no instances given as param an not up to date
		defaultSelected := instanceGivenAsParam || (instance.Workflow != nil && notUpToDate && len(moperations) == 0)

		opts[i] = cli.CustomMultiSelectOption{
			Value:   key,
			Info:    info,
			Default: defaultSelected,
		}
		values[key] = instance
		i++
	}

	var results []string
	if len(opts) > 0 {
		prompt := cli.NewCustomMultiSelect("Select template's instances that you want to update", opts...)
		if err := survey.AskOne(prompt, &results, nil); err != nil {
			return err
		}
	}

	// for all selected instances, add it to operations map
	for i := range results {
		key := results[i]
		if _, ok := moperations[key]; !ok {
			moperations[key] = sdk.WorkflowTemplateBulkOperation{
				Request: mwtis[key].Request,
			}
		}
	}

	return nil
}

func templateBulkRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}

	// validate data from file
	filePath := v.GetString("instances-file")
	wtFromFile, fileOperations, err := templateExtractAndValidateFileParams(filePath)
	if err != nil {
		return err
	}
	if wtFromFile != nil {
		wt = wtFromFile
	}

	// if no template found for workflow or no instance, suggest one
	if wt == nil {
		if v.GetBool("no-interactive") {
			return fmt.Errorf("you should give a template path")
		}
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	// validate instances format
	instanceKeys := v.GetStringArray("instances")
	minstances, err := templateExtractAndValidateInstances(instanceKeys)
	if err != nil {
		return err
	}

	// validate params format
	rawParams := v.GetStringArray("params")
	params, err := templateExtractAndValidateParams(rawParams)
	if err != nil {
		return err
	}

	// get all existings template instances
	wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
	if err != nil {
		return err
	}
	// filter as code workflow
	wtisFiltered := make([]sdk.WorkflowTemplateInstance, 0, len(wtis))
	for i := range wtis {
		if wtis[i].Workflow.FromRepository == "" {
			wtisFiltered = append(wtisFiltered, wtis[i])
		}
	}
	wtis = wtisFiltered

	mwtis := make(map[string]sdk.WorkflowTemplateInstance, len(wtis))
	for _, i := range wtis {
		mwtis[i.Key()] = i
	}

	moperations := templateInitOperationFromParams(mwtis, fileOperations, minstances, params)

	// set detach for existing operations
	rawDetach := v.GetStringArray("detach")
	for _, d := range rawDetach {
		if _, ok := moperations[d]; ok {
			o := moperations[d]
			o.Request.Detached = true
			moperations[d] = o
		}
	}

	// ask interactively for params if prompt not disabled
	if !v.GetBool("no-interactive") {
		sort.Slice(wtis, func(i, j int) bool { return wtis[i].Key() < wtis[j].Key() })
		if err := templateAskForInstances(wt, mwtis, minstances, moperations); err != nil {
			return err
		}

		// init map of projects and project repositories to prevent multiple api calls
		mprojects := make(map[string]*sdk.Project, len(mwtis))
		for _, wti := range mwtis {
			mprojects[wti.Project.Key] = wti.Project
		}
		projectRepositories := make(map[string][]string)

		for operationKey, operation := range moperations {
			// check if some params are missing for current operation
			var paramMissing bool
			for _, p := range wt.Parameters {
				if _, ok := operation.Request.Parameters[p.Key]; !ok {
					paramMissing = true
					break
				}
			}

			if paramMissing {
				// get project from map if exists else from api
				if _, ok := mprojects[operationKey]; !ok {
					p, err := client.ProjectGet(operation.Request.ProjectKey)
					if err != nil {
						return err
					}
					mprojects[p.Key] = p
				}
				project := mprojects[operationKey]

				// for each param not already in previous request ask for the value
				for _, p := range wt.Parameters {
					if _, ok := operation.Request.Parameters[p.Key]; !ok {
						label := fmt.Sprintf("Value for param '%s' on '%s' (type: %s, required: %t)", p.Key, operationKey, p.Type, p.Required)

						var value string
						switch p.Type {
						case sdk.ParameterTypeRepository:
							// get the project and its repositories if not already loaded
							if _, ok := projectRepositories[project.Key]; !ok {
								for _, vcs := range project.VCSServers {
									rs, err := client.RepositoriesList(project.Key, vcs.Name, false)
									if err != nil {
										return err
									}
									for _, r := range rs {
										projectRepositories[project.Key] = append(projectRepositories[project.Key],
											fmt.Sprintf("%s/%s", vcs.Name, r.Slug))
									}
								}
							}

							// ask to choose a repository, if only one ask to, if no repo found ask for value
							lengthRepo := len(projectRepositories[project.Key])
							if lengthRepo > 1 {
								if err := survey.AskOne(&survey.Select{
									Message: label,
									Options: projectRepositories[project.Key],
								}, &value, nil); err != nil {
									return err
								}
							} else if lengthRepo == 1 {
								var result bool
								if err := survey.AskOne(&survey.Confirm{
									Message: fmt.Sprintf("Set value to '%s' for param '%s' on '%s'", projectRepositories[project.Key][0], p.Key, operationKey),
									Default: true,
								}, &result, nil); err != nil {
									return err
								}
								if result {
									value = projectRepositories[project.Key][0]
								}
							}
							if value == "" {
								if err := survey.AskOne(&survey.Input{Message: label}, &value, nil); err != nil {
									return err
								}
							}
						case sdk.ParameterTypeBoolean:
							var result bool
							if err := survey.AskOne(&survey.Confirm{
								Message: fmt.Sprintf("Set value to 'true' for param '%s' on '%s'", p.Key, operationKey),
								Default: true,
							}, &result, nil); err != nil {
								return err
							}
							value = fmt.Sprintf("%t", result)
						default:
							if err := survey.AskOne(&survey.Input{Message: label}, &value, nil); err != nil {
								return err
							}
						}

						operation.Request.Parameters[p.Key] = value
					}
				}
			}
		}
	}

	if len(moperations) == 0 {
		fmt.Printf("Nothing to do")
		return nil
	}

	// send bulk request
	b := sdk.WorkflowTemplateBulk{Operations: make([]sdk.WorkflowTemplateBulkOperation, len(moperations))}
	i := 0
	for _, o := range moperations {
		b.Operations[i] = o
		i++
	}

	res, err := client.TemplateBulk(wt.Group.Name, wt.Slug, b)
	if err != nil {
		return err
	}

	fmt.Printf("Bulk request with id %d successfully created for template %s/%s with %d operations\n", res.ID, wt.Group.Name, wt.Slug, len(res.Operations))

	if v.GetBool("track") {
		var currentDisplay = new(cli.Display)
		currentDisplay.Printf("Looking for bulk %d...\n", b.ID)
		currentDisplay.Do(context.Background())

		for {
			res, err = client.TemplateGetBulk(wt.Group.Name, wt.Slug, res.ID)
			if err != nil {
				return err
			}

			var out string
			for _, o := range res.Operations {
				var status string
				switch o.Status {
				case sdk.OperationStatusPending:
					status = cli.Blue("pending")
				case sdk.OperationStatusProcessing:
					status = cli.Yellow("processing")
				case sdk.OperationStatusDone:
					status = cli.Green("done")
				case sdk.OperationStatusError:
					status = cli.Red("error")
				}
				out += fmt.Sprintf("%s/%s -> %s %s\n", o.Request.ProjectKey, o.Request.WorkflowName, status, o.Error)
			}

			currentDisplay.Printf(out)

			time.Sleep(500 * time.Millisecond)
			if res.IsDone() {
				break
			}
		}
	}

	return nil
}
