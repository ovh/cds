package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/jfrog/jfrog-client-go/utils/log"

	"github.com/ovh/cds/contrib/grpcplugins"
	art "github.com/ovh/cds/contrib/integrations/artifactory"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

/*
This plugin have to be used as a promote plugin

Artifactory promote plugin must configured as following:
	name: artifactory-promote-plugin
	type: integration
	author: "Steven Guiheux"
	description: "OVH Artifactory Promote Plugin"

$ cdsctl admin plugins import artifactory-promote-plugin.yml

Build the present binaries and import in CDS:
	os: linux
	arch: amd64
	cmd: <path-to-binary-file>

$ cdsctl admin plugins binary-add artifactory-promote-plugin artifactory-promote-plugin-bin.yml <path-to-binary-file>
*/

type artifactoryPromotePlugin struct {
	integrationplugin.Common
}

func (e *artifactoryPromotePlugin) Manifest(_ context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "OVH Artifactory Promote Plugin",
		Author:      "Steven Guiheux",
		Description: "OVH Artifactory Promote Plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *artifactoryPromotePlugin) Run(ctx context.Context, opts *integrationplugin.RunQuery) (*integrationplugin.RunResult, error) {
	artifactoryURL := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigURL)]
	token := opts.GetOptions()[fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken)]

	artifactList := opts.GetOptions()["artifacts"]

	runResult, err := grpcplugins.GetRunResults(e.HTTPPort)
	if err != nil {
		return fail("unable to list run results: %v", err)
	}

	log.SetLogger(log.NewLogger(log.INFO, os.Stdout))

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	artiClient, err := art.CreateArtifactoryClient(ctx, artifactoryURL, token)
	if err != nil {
		return fail("unable to create artifactory client: %v", err)
	}

	artSplit := strings.Split(artifactList, ",")
	artRegs := make([]*regexp.Regexp, 0, len(artSplit))
	for _, arti := range artSplit {
		r, err := regexp.Compile(arti)
		if err != nil {
			return fail("unable compile regexp in artifact list: %v", err)
		}
		artRegs = append(artRegs, r)
	}

	for _, r := range runResult {
		rData, err := r.GetArtifactManager()
		if err != nil {
			return fail("unable to read result %s: %v", r.ID, err)
		}
		skip := true
		for _, reg := range artRegs {
			if reg.MatchString(rData.Name) {
				skip = false
				break
			}
		}
		if skip {
			continue
		}
		if r.DataSync == nil {
			return fail("unable to find an existing promotion for result %s", r.ID)
		}
		latestPromotion := r.DataSync.LatestPromotionOrRelease()
		if latestPromotion == nil {
			return fail("unable to find an existing promotion for result %s", r.ID)
		}
		switch rData.RepoType {
		case "docker":
			if err := art.PromoteDockerImage(artiClient, rData, latestPromotion.FromMaturity, latestPromotion.ToMaturity); err != nil {
				return fail("unable to promote docker image: %s: %v", rData.Name+"-"+latestPromotion.ToMaturity, err)
			}
		default:
			if err := art.PromoteFile(artiClient, rData, latestPromotion.FromMaturity, latestPromotion.ToMaturity); err != nil {
				return fail("unable to promote file: %s: %v", rData.Name, err)
			}
		}
	}
	return &integrationplugin.RunResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	e := artifactoryPromotePlugin{}
	if err := integrationplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
}

func fail(format string, args ...interface{}) (*integrationplugin.RunResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &integrationplugin.RunResult{
		Details: msg,
		Status:  sdk.StatusFail,
	}, nil
}
