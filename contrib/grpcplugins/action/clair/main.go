package main

import (
	"context"
	"fmt"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/viper"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/clair"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/config"
	"github.com/ovh/cds/contrib/grpcplugins/action/clair/clairctl/docker"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build clair
$ make publish clair
*/

type clairActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *clairActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-clair",
		Author:      "Steven GUIHEUX <steven.guiheux@corp.ovh.com>",
		Description: `This is a plugin to run clair analysis`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *clairActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	// get clair configuration
	fmt.Printf("Retrieve clair configuration...")
	serv, err := grpcplugins.GetExternalServices(actPlugin.HTTPPort, "clair")
	if err != nil {
		return fail("Unable to get clair configuration: %s", err)
	}
	viper.Set("clair.uri", serv.URL)
	viper.Set("clair.port", serv.Port)
	viper.Set("clair.healthPort", serv.HealthPort)
	viper.Set("clair.report.path", ".")
	viper.Set("clair.report.format", "json")
	clair.Config()

	dockerImage := q.GetOptions()["image"]

	fmt.Printf("Pushing image %s into clair\n", dockerImage)
	image, manifest, err := pushImage(dockerImage)
	if err != nil {
		return fail("Unable to push image on Clair: %s", err)
	}

	fmt.Printf("Running analysis\n")
	analysis, err := clair.Analyze(image, manifest)
	if err != nil {
		return fail("Error on running analysis with Clair: %v", err)
	}

	fmt.Printf("Creating report")

	var vulnerabilities []sdk.Vulnerability
	summary := make(map[string]int64)
	if analysis.MostRecentLayer().Layer != nil {
		for _, feat := range analysis.MostRecentLayer().Layer.Features {
			for _, vuln := range feat.Vulnerabilities {
				v := sdk.Vulnerability{
					Version:     feat.Version,
					Component:   feat.Name,
					Description: vuln.Description,
					Link:        vuln.Link,
					FixIn:       vuln.FixedBy,
					Severity:    sdk.ToVulnerabilitySeverity(vuln.Severity),
					Origin:      feat.AddedBy,
					CVE:         vuln.Name,
					Title:       fmt.Sprintf("%s %s", feat.Name, feat.Version),
				}
				vulnerabilities = append(vulnerabilities, v)

				count := summary[v.Severity]
				summary[v.Severity] = count + 1
			}
		}
	}

	report := sdk.VulnerabilityWorkerReport{
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
		Type:            "docker",
	}
	if err := grpcplugins.SendVulnerabilityReport(actPlugin.HTTPPort, report); err != nil {
		return fail("Unable to send report: %s", err)
	}
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *clairActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := clairActionPlugin{}
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

func pushImage(dockerImage string) (reference.NamedTagged, distribution.Manifest, error) {
	config.ImageName = dockerImage
	image, manifest, err := docker.RetrieveManifest(config.ImageName, true)
	if err != nil {
		return image, manifest, fmt.Errorf("pushImage> unable to retrieve manifest: %v", err)
	}

	if err := clair.Push(image, manifest); err != nil {
		if err != nil {
			return image, manifest, fmt.Errorf("pushImage> unable to push image %q: %v", image.String(), err)
		}
	}
	return image, manifest, nil
}
