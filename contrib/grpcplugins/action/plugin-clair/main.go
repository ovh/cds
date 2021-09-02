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
	servs, err := grpcplugins.GetServices(actPlugin.HTTPPort, "clair")
	if err != nil {
		return actionplugin.Fail("Unable to get clair configuration: %s", err)
	}
	serv := servs[0]
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
		return actionplugin.Fail("Unable to push image on Clair: %s", err)
	}

	fmt.Printf("Running analysis\n")
	analysis, err := clair.Analyze(image, manifest)
	if err != nil {
		return actionplugin.Fail("Error on running analysis with Clair: %v", err)
	}

	fmt.Printf("Creating report")

	var vulnerabilities []sdk.Vulnerability
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
			}
		}
	}

	report := sdk.VulnerabilityWorkerReport{
		Vulnerabilities: vulnerabilities,
		Type:            "docker",
	}
	if err := grpcplugins.SendVulnerabilityReport(actPlugin.HTTPPort, report); err != nil {
		return actionplugin.Fail("Unable to send report: %s", err)
	}
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := clairActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
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
