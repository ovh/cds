package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

//NpmAuditParserPlugin is a plugin to run clair analysis
type NpmAuditParserPlugin struct {
	plugin.Common
}

//Name return plugin name. It must me the same as the binary file
func (d NpmAuditParserPlugin) Name() string {
	return "plugin-npm-audit-parser"
}

//Description explains the purpose of the plugin
func (d NpmAuditParserPlugin) Description() string {
	return "This is a plugin to parse npm audit report"
}

//Author of the plugin
func (d NpmAuditParserPlugin) Author() string {
	return "Steven GUIHEUX <steven.guiheux@corp.ovh.com>"
}

// Parameters return parameters description
func (d NpmAuditParserPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("file", plugin.StringParameter, "path to audit file", "")
	return params
}

// Run execute the action
func (d NpmAuditParserPlugin) Run(j plugin.IJob) plugin.Result {
	file := j.Arguments().Get("file")
	if file == "" {
		_ = plugin.SendLog(j, "File parameter must not be empty")
		return plugin.Fail
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		_ = plugin.SendLog(j, "Unable to read file %s: %v", file, err)
		return plugin.Fail
	}
	var npmAudit NpmAudit
	if err := json.Unmarshal(b, &npmAudit); err != nil {
		_ = plugin.SendLog(j, "Unable to read npm report: %v", err)
		return plugin.Fail
	}

	var report sdk.VulnerabilityWorkerReport
	summary := make(map[string]int64)
	for _, a := range npmAudit.Advisories {
		for _, f := range a.Findings {
			if len(a.CVES) > 0 {
				for _, c := range a.CVES {
					v := sdk.Vulnerability{
						Component:   a.ModuleName,
						CVE:         c,
						Description: a.Overview,
						FixIn:       a.PatchedVersions,
						Link:        a.URL,
						Origin:      strings.Join(f.Paths, "\n"),
						Severity:    sdk.ToVulnerabilitySeverity(a.Severity),
						Title:       a.Title,
						Version:     f.Version,
					}
					report.Vulnerabilities = append(report.Vulnerabilities, v)
					count := summary[v.Severity]
					summary[v.Severity] = count + 1
				}
			} else {
				v := sdk.Vulnerability{
					Component:   a.ModuleName,
					CVE:         a.CWE,
					Description: a.Overview,
					FixIn:       a.PatchedVersions,
					Link:        a.URL,
					Origin:      strings.Join(f.Paths, "\n"),
					Severity:    sdk.ToVulnerabilitySeverity(a.Severity),
					Title:       a.Title,
					Version:     f.Version,
				}
				report.Vulnerabilities = append(report.Vulnerabilities, v)
				count := summary[v.Severity]
				summary[v.Severity] = count + 1
			}

		}
	}
	report.Summary = summary
	if err := plugin.SendVulnerabilityReport(j, report); err != nil {
		_ = plugin.SendLog(j, "Unable to send report: %s", err)
		return plugin.Fail
	}
	return plugin.Success
}

func main() { // nolint
	plugin.Main(&NpmAuditParserPlugin{})
}
