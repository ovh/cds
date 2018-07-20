package plugin_clair

import (
	"fmt"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/jgsqware/clairctl/clair"
	"github.com/jgsqware/clairctl/config"
	"github.com/jgsqware/clairctl/docker"

	"github.com/json-iterator/go"
	"github.com/ovh/cds/sdk/plugin"
)

//ClairPlugin is a plugin to run clair analysis
type ClairPlugin struct {
	plugin.Common
}

//Name return plugin name. It must me the same as the binary file
func (d ClairPlugin) Name() string {
	return "plugin-clairctl"
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
	dockerImage := a.Arguments().Get("image")

	image, manifest, err := pushImage(dockerImage)
	if err != nil {
		plugin.SendLog(a, "Unable to push image on Clair: %s", err)
		return plugin.Fail
	}

	analysis := clair.Analyze(image, manifest)

	img := strings.Replace(analysis.ImageName, "/", "-", -1)
	if analysis.Tag != "" {
		img += "-" + analysis.Tag
	}

	json, err := jsoniter.Marshal(analysis)
	if err != nil {
		plugin.SendLog(a, "Unable to push image on Clair: %s", err)
		return plugin.Fail
	}
	plugin.SendLog(a, "Report: %s", string(json))
	return plugin.Success
}

func main() {
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
