package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ovh/cds/cli"

	"gopkg.in/yaml.v2"
)

var templateInstancesCmd = cli.Command{
	Name:    "instances",
	Aliases: []string{"instances"},
	Short:   "Get instances for a CDS workflow template",
	Example: "cdsctl template instances group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
}

func templateInstancesRun(v cli.Values) (cli.ListResult, error) {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return nil, err
	}
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return nil, err
		}
	}

	wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
	if err != nil {
		return nil, err
	}

	type TemplateInstanceDisplay struct {
		ID       int64  `cli:"id,key"`
		Created  string `cli:"created"`
		Project  string `cli:"project"`
		Workflow string `cli:"workflow"`
		Params   string `cli:"params"`
		Version  int64  `cli:"version"`
		UpToDate bool   `cli:"uptodate"`
	}

	tids := make([]TemplateInstanceDisplay, len(wtis))
	for i := range wtis {
		tids[i].ID = wtis[i].ID
		tids[i].Created = fmt.Sprintf("On %s by %s", wtis[i].FirstAudit.Created.Format(time.RFC3339),
			wtis[i].FirstAudit.AuditCommon.TriggeredBy)
		tids[i].Project = wtis[i].Project.Name
		if wtis[i].Workflow != nil {
			tids[i].Workflow = wtis[i].Workflow.Name
		} else {
			tids[i].Workflow = fmt.Sprintf("%s (not imported)", wtis[i].WorkflowName)
		}
		for k, v := range wtis[i].Request.Parameters {
			tids[i].Params = fmt.Sprintf("%s%s:%s\n", tids[i].Params, k, v)
		}
		tids[i].Version = wtis[i].WorkflowTemplateVersion
		tids[i].UpToDate = wtis[i].WorkflowTemplateVersion == wt.Version
	}

	return cli.AsListResult(tids), nil
}

var templateInstancesExportCmd = cli.Command{
	Name:    "export",
	Short:   "Get instances for a CDS workflow template as yaml file",
	Example: "cdsctl template instances group-name/template-slug",
	OptionalArgs: []cli.Arg{
		{Name: "template-path"},
	},
	Flags: []cli.Flag{
		{
			Type:    cli.FlagArray,
			Name:    "filter-params",
			Usage:   "Specify filter on params for template like --filter-params paramKey=paramValue, wil return only instances that have params that match.",
			Default: "",
		},
	},
}

func templateInstancesExportRun(v cli.Values) error {
	wt, err := getTemplateFromCLI(v)
	if err != nil {
		return err
	}
	if wt == nil {
		wt, err = suggestTemplate()
		if err != nil {
			return err
		}
	}

	wtis, err := client.TemplateGetInstances(wt.Group.Name, wt.Slug)
	if err != nil {
		return err
	}

	filterParams := make(map[string]string)
	filterParamPairs := v.GetStringArray("filter-params")
	for _, p := range filterParamPairs {
		ps := strings.Split(p, "=")
		if len(ps) < 2 {
			return fmt.Errorf("Invalid given param %s", ps[0])
		}
		filterParams[ps[0]] = strings.Join(ps[1:], "=")
	}

	var f templateBulkFile
	f.TemplatePath = wt.Path()

	for _, wti := range wtis {
		filterMatch := true
		for k, v := range filterParams {
			if value, ok := wti.Request.Parameters[k]; !ok || value != v {
				filterMatch = false
				break
			}
		}
		if !filterMatch {
			continue
		}

		params := []string{}
		for k, v := range wti.Request.Parameters {
			params = append(params, k+"="+v)
		}
		f.Instances = append(f.Instances, templateBulkFileInstance{
			WorkflowPath: wti.Project.Key + "/" + wti.Request.WorkflowName,
			Parameters:   params,
		})
	}

	b, err := yaml.Marshal(f)
	if err != nil {
		return fmt.Errorf("unable to marshal: %v", err)
	}

	fmt.Println(string(b))

	return nil
}
