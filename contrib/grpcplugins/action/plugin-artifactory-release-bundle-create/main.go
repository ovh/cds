package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-cli-core/artifactory/spec"
	"github.com/jfrog/jfrog-cli-core/utils/coreutils"
	"github.com/jfrog/jfrog-client-go/distribution/services"
	distributionServicesUtils "github.com/jfrog/jfrog-client-go/distribution/services/utils"
	"github.com/jfrog/jfrog-client-go/utils/log"

	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build plugin-artifactory-release-bundle-create
$ make publish plugin-artifactory-release-bundle-create
*/

type artifactoryReleaseBundleCreatePlugin struct {
	actionplugin.Common
}

func (actPlugin *artifactoryReleaseBundleCreatePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-artifactory-release-bundle-create",
		Author:      "François Samin <francois.samin@corp.ovh.com>",
		Description: `This action creates and sign a release bundle from a yaml specification`,
		Version:     sdk.VERSION,
	}, nil
}

func unmarshalSpecFile(content []byte) (s *spec.SpecFiles, err error) {
	defer func() {
		if r := recover(); r != nil {
			s = nil
			err = fmt.Errorf("invalid given specification")
		}
	}()

	var schema spec.SpecFiles
	if err := yaml.Unmarshal(content, &schema); err != nil {
		return nil, fmt.Errorf("invalid given specification: %v", err)
	}

	if err := spec.ValidateSpec(schema.Files, false, true, false); err != nil {
		return nil, fmt.Errorf("invalid release bundle spec: %v", err)
	}

	return &schema, nil
}

func (actPlugin *artifactoryReleaseBundleCreatePlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))

	name := q.GetOptions()["name"]
	version := q.GetOptions()["version"]
	description := q.GetOptions()["description"]
	releaseNotes := q.GetOptions()["release_notes"]
	specification := q.GetOptions()["specification"]
	variables := q.GetOptions()["variables"]
	url := q.GetOptions()["url"]
	token := q.GetOptions()[q.GetOptions()["token_variable"]]

	if url == "" {
		url = q.GetOptions()["cds.integration.artifact_manager.url"]
		token = q.GetOptions()["cds.integration.artifact_manager.release.token"]
	}

	if url == "" {
		return actionplugin.Fail("missing Artifactory URL")
	}
	if token == "" {
		return actionplugin.Fail("missing Artifactory Distribution Token")
	}

	var specVars map[string]string

	fmt.Printf("Preparing release bundle %q version %q\n", name, version)
	if err := yaml.Unmarshal([]byte(variables), &specVars); err != nil {
		return actionplugin.Fail("invalid given variables: %v", err)
	}

	var content = []byte(specification)
	if len(specVars) > 0 {
		content = coreutils.ReplaceVars(content, specVars)
	}

	var schema, err = unmarshalSpecFile(content)
	if err != nil {
		return actionplugin.Fail("%v", err)
	}

	var releaseBundleParams = distributionServicesUtils.NewReleaseBundleParams(name, version)
	releaseBundleParams.SignImmediately = true
	releaseBundleParams.Description = description
	releaseBundleParams.ReleaseNotes = releaseNotes
	releaseBundleParams.ReleaseNotesSyntax = distributionServicesUtils.Markdown
	for _, spec := range schema.Files {
		p, err := spec.ToArtifactoryCommonParams()
		if err != nil {
			return actionplugin.Fail("invalid spec file: %v", err)
		}
		releaseBundleParams.SpecFiles = append(releaseBundleParams.SpecFiles, p)
	}

	var params = services.CreateReleaseBundleParams{ReleaseBundleParams: releaseBundleParams}
	fmt.Printf("Connecting to %s...\n", url)
	distriClient, err := art.CreateDistributionClient(url, token)
	if err != nil {
		return actionplugin.Fail("unable to create distribution client: %v", err)
	}

	btes, _ := json.MarshalIndent(params, "  ", "  ")
	fmt.Println(string(btes))

	fmt.Printf("Creating release %s %s\n", params.Name, params.Version)
	if _, err := distriClient.CreateReleaseBundle(params); err != nil {
		return actionplugin.Fail("unable to create release bundle: %v", err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := artifactoryReleaseBundleCreatePlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
