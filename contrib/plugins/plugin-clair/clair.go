package main

import (
	"fmt"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/jgsqware/clairctl/clair"
	"github.com/jgsqware/clairctl/config"
	"github.com/jgsqware/clairctl/docker"
	"github.com/spf13/viper"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

//ClairPlugin is a plugin to run clair analysis
type ClairPlugin struct {
	plugin.Common
}

//Name return plugin name. It must me the same as the binary file
func (d ClairPlugin) Name() string {
	return "plugin-clair"
}

//Description explains the purpose of the plugin
func (d ClairPlugin) Description() string {
	return "This is a plugin to run clair analysis"
}

//Author of the plugin
func (d ClairPlugin) Author() string {
	return "Steven GUIHEUX <steven.guiheux@corp.ovh.com>"
}

// Parameters return parameters description
func (d ClairPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("image", plugin.StringParameter, "Image to analyze", "")
	return params
}

// Run execute the action
func (d ClairPlugin) Run(a plugin.IJob) plugin.Result {
	// get clair configuration
	_ = plugin.SendLog(a, "Retrieve clair configuration...")
	serv, err := plugin.GetExternalServices(a, "clair")
	if err != nil {
		_ = plugin.SendLog(a, "Unable to get clair configuration: %s", err)
		return plugin.Fail
	}
	viper.Set("clair.uri", serv.URL)
	viper.Set("clair.port", serv.Port)
	viper.Set("clair.healthPort", serv.HealthPort)
	viper.Set("clair.report.path", ".")
	viper.Set("clair.report.format", "json")
	clair.Config()

	dockerImage := a.Arguments().Get("image")

	_ = plugin.SendLog(a, "Pushing image %s into clair", dockerImage)
	image, manifest, err := pushImage(dockerImage)
	if err != nil {
		_ = plugin.SendLog(a, "Unable to push image on Clair: %s", err)
		return plugin.Fail
	}

	_ = plugin.SendLog(a, "Running analysis")
	analysis := clair.Analyze(image, manifest)

	_ = plugin.SendLog(a, "Creating report")

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

	report := sdk.VulnerabilityReport{
		Vulnerabilities: vulnerabilities,
		Summary:         summary,
	}
	if err := plugin.SendVulnerabilityReport(a, report); err != nil {
		_ = plugin.SendLog(a, "Unable to send report: %s", err)
		return plugin.Fail
	}
	return plugin.Success
}

func main() { // nolint
	plugin.Main(&ClairPlugin{})
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
