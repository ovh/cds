package main

import (
	"fmt"
	"reflect"

	"github.com/ovh/cds/sdk"

	"github.com/AlecAivazis/survey"

	"github.com/ovh/cds/cli"
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
			Kind:      reflect.String,
			Name:      "output-dir",
			ShortHand: "d",
			Usage:     "Output directory",
			Default:   ".cds",
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

		opts := make([]cli.CustomMultiSelectOption, len(wtis))
		values := make(map[string]sdk.WorkflowTemplateInstance, len(wtis))
		for i := range wtis {
			if wtis[i].Workflow != nil {
				notUpToDate := wtis[i].WorkflowTemplateVersion < wt.Version

				var info string
				if notUpToDate {
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

				key := fmt.Sprintf("%s/%s", wtis[i].Project.Key, wtis[i].Workflow.Name)
				opts[i] = cli.CustomMultiSelectOption{
					Value:   key,
					Info:    info,
					Default: true,
				}
				values[key] = wtis[i]
			}
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

		projectRepositories := map[string][]string{}
		for i := range results {
			wti := values[results[i]]

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
							value = fmt.Sprintf("%T", v)
						} else {
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
						value = fmt.Sprintf("%T", v)
					default:
						if err := survey.AskOne(&survey.Input{Message: label}, &value, nil); err != nil {
							return err
						}
					}

					wti.Request.Parameters[p.Key] = value
				}
			}
		}

		// TODO send bulk request
	}

	return nil
}
