package main

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/AlecAivazis/survey"

	"github.com/ovh/cds/cli"
	"github.com/ovh/cds/sdk"
)

var templateBulkCmd = cli.Command{
	Name:    "bulk",
	Short:   "Bulk apply CDS workflow template and push all given workflows",
	Example: "cdsctl template bulk group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Kind:      reflect.Slice,
			Name:      "params",
			ShortHand: "p",
			Usage:     "Specify params for template",
			Default:   "",
		},
		{
			Kind:      reflect.Bool,
			Name:      "no-interactive",
			ShortHand: "n",
			Usage:     "Set to not ask interactively for params",
		},
		{
			Kind:  reflect.Bool,
			Name:  "track",
			Usage: "Wait the bulk to be over",
		},
	},
}

func templateBulkRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}

	// if no template found for workflow or no instance, suggest one
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	// ask interactively for params if prompt not disabled
	if !v.GetBool("no-interactive") {
		// get all existings template instances
		wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
		if err != nil {
			return err
		}

		sort.Slice(wtis, func(i, j int) bool { return wtis[i].Key() < wtis[j].Key() })

		opts := []cli.CustomMultiSelectOption{}
		values := make(map[string]sdk.WorkflowTemplateInstance, len(wtis))
		for i := range wtis {
			notUpToDate := wtis[i].WorkflowTemplateVersion < wt.Version

			var info string
			if wtis[i].Workflow == nil {
				info = cli.Yellow("not imported")
			} else if notUpToDate {
				info = cli.Red("not up to date")
			} else {
				info = cli.Green("up to date")
			}

			var paramMissing bool
			for _, p := range wt.Parameters {
				if _, ok := wtis[i].Request.Parameters[p.Key]; !ok {
					paramMissing = true
					break
				}
			}

			if notUpToDate {
				if paramMissing {
					info = fmt.Sprintf("%s - %s", info, cli.Red("needs parameters to apply"))
				} else {
					info = fmt.Sprintf("%s - %s", info, cli.Green("can apply automatically"))
				}
			}

			key := wtis[i].Key()
			opts = append(opts, cli.CustomMultiSelectOption{
				Value:   key,
				Info:    info,
				Default: wtis[i].Workflow != nil && notUpToDate,
			})
			values[key] = wtis[i]
		}

		results := []string{}
		prompt := &cli.CustomMultiSelect{
			Message: "Select template's instances that you want to update",
			Options: opts,
		}
		prompt.Init()
		if err := survey.AskOne(prompt, &results, nil); err != nil {
			return err
		}

		operations := make([]sdk.WorkflowTemplateBulkOperation, len(results))

		projectRepositories := map[string][]string{}
		for i := range results {
			wti := values[results[i]]

			operations[i].Request = wti.Request

			// for each param not already in previous request ask for the value
			for _, p := range wt.Parameters {
				if _, ok := wti.Request.Parameters[p.Key]; !ok {
					label := fmt.Sprintf("Value for param '%s' on '%s' (type: %s, required: %t)", p.Key, results[i], p.Type, p.Required)

					var value string
					switch p.Type {
					case sdk.ParameterTypeRepository:
						// get the project and its repositories if not already loaded
						if _, ok := projectRepositories[wti.Project.Key]; !ok {
							project, err := client.ProjectGet(wti.Project.Key)
							if err != nil {
								return err
							}

							for _, vcs := range project.VCSServers {
								rs, err := client.RepositoriesList(project.Key, vcs.Name)
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
						lengthRepo := len(projectRepositories[wti.Project.Key])
						if lengthRepo > 1 {
							if err := survey.AskOne(&survey.Select{
								Message: label,
								Options: projectRepositories[wti.Project.Key],
							}, &value, nil); err != nil {
								return err
							}
						} else if lengthRepo == 1 {
							var result bool
							if err := survey.AskOne(&survey.Confirm{
								Message: fmt.Sprintf("Set value to '%s' for param '%s' on '%s'", projectRepositories[wti.Project.Key][0], p.Key, results[i]),
								Default: true,
							}, &result, nil); err != nil {
								return err
							}
							if result {
								value = projectRepositories[wti.Project.Key][0]
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
							Message: fmt.Sprintf("Set value to 'true' for param '%s' on '%s'", p.Key, results[i]),
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

					operations[i].Request.Parameters[p.Key] = value
				}
			}
		}

		// send bulk request
		b := sdk.WorkflowTemplateBulk{Operations: operations}

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
	}

	return nil
}
