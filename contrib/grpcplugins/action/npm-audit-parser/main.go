package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build npm-audit-parser
$ make publish npm-audit-parser
*/

type npmAuditParserActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *npmAuditParserActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-npm-audit-parser",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: "This is a plugin to parse npm audit report",
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *npmAuditParserActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	file := q.GetOptions()["file"]
	if file == "" {
		return fail("File parameter must not be empty")
	}
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return fail("Unable to read file %s: %v", file, err)
	}
	var npmAudit NpmAudit
	if err := json.Unmarshal(b, &npmAudit); err != nil {
		return fail("Unable to read npm report: %v", err)
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
	report.Type = "js"
	report.Summary = summary
	if err := grpcplugins.SendVulnerabilityReport(actPlugin.HTTPPort, report); err != nil {
		return fail("Unable to send report: %s", err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *npmAuditParserActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := npmAuditParserActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}

func fail(format string, args ...interface{}) (*actionplugin.ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &actionplugin.ActionResult{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
}
