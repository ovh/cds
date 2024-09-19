package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-units"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/client"
	"github.com/pkg/errors"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type dockerPushPlugin struct {
	actionplugin.Common
}

func main() {
	actPlugin := dockerPushPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}

func (actPlugin *dockerPushPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "dockerPush",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Push an image docker on a docker registry",
		Version:     sdk.VERSION,
	}, nil
}

func (p *dockerPushPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	image := q.GetOptions()["image"]
	tags := q.GetOptions()["tags"]
	registry := q.GetOptions()["registry"]
	auth := q.GetOptions()["registryAuth"]

	tags = strings.Replace(tags, " ", ",", -1) // If tags are separated by <space>
	tags = strings.Replace(tags, ";", ",", -1) // If tags are separated by <semicolon>

	var tagSlice []string
	if len(tags) > 0 {
		tagSlice = strings.Split(tags, ",")
	}

	if !strings.ContainsRune(image, ':') { // Latest is the default tag
		image = image + ":latest"
	}

	if err := p.perform(ctx, image, tagSlice, registry, auth); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
	}
	return stream.Send(res)

}

// Run implements actionplugin.ActionPluginServer.
func (actPlugin *dockerPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *dockerPushPlugin) perform(ctx context.Context, image string, tags []string, registry, registryAuth string) error {
	if image == "" {
		return sdk.Errorf("wrong usage: <image> parameter should not be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return sdk.Errorf("unable to get instantiate docker client: %v", err)
	}

	imageSummaries, err := cli.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return sdk.Errorf("unable to get docker image %q: %v", image, err)
	}

	images := []grpcplugins.Img{}
	for _, image := range imageSummaries {
		repository := "<none>"
		tag := "<none>"
		duration := HumanDuration(image.Created)
		size := HumanSize(image.Size)
		if len(image.RepoTags) > 0 {
			for _, rt := range image.RepoTags {
				splitted := strings.Split(rt, ":")
				repository = splitted[0]
				tag = splitted[1]
				images = append(images, grpcplugins.Img{Repository: repository, Tag: tag, ImageID: image.ID[7:19], Created: duration, Size: size})
			}
		} else if len(image.RepoDigests) > 0 {
			repository = strings.Split(image.RepoDigests[0], "@")[0]
			images = append(images, grpcplugins.Img{Repository: repository, Tag: tag, ImageID: image.ID[7:19], Created: duration, Size: size})
		}
	}

	var imgFound *grpcplugins.Img
	for i := range images {
		if images[i].Repository+":"+images[i].Tag == image {
			imgFound = &images[i]
			break
		}
	}

	if imgFound == nil {
		return sdk.Errorf("image %q not found", image)
	}

	if len(tags) == 0 { // If no tag is provided, keep the actual tag
		tags = []string{imgFound.Tag}
	}

	for _, tag := range tags {
		result, d, err := actPlugin.performImage(ctx, cli, image, imgFound, registry, registryAuth, strings.TrimSpace(tag))
		if err != nil {
			return err
		}
		grpcplugins.Successf(&actPlugin.Common, "Image %s pushed in %.3fs", result.Name(), d.Seconds())
	}

	return nil
}

func (actPlugin *dockerPushPlugin) performImage(ctx context.Context, cli *client.Client, source string, img *grpcplugins.Img, registryURL string, registryAuth string, tag string) (*sdk.V2WorkflowRunResult, time.Duration, error) {
	var t0 = time.Now()
	// Create run result at status "pending"
	var runResultRequest = workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeDocker,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail:   grpcplugins.ComputeRunResultDockerDetail(source, *img),
		},
	}

	response, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest)
	if err != nil {
		return nil, time.Since(t0), err
	}

	result := response.RunResult

	jobCtx, err := grpcplugins.GetJobContext(ctx, &actPlugin.Common)
	if err != nil {
		return nil, time.Since(t0), err
	}

	var destination string
	// Upload the file to an artifactory or the docker registry
	switch {
	case result.ArtifactManagerIntegrationName != nil:
		if jobCtx.Integrations == nil || jobCtx.Integrations.ArtifactManager.Name == "" {
			return nil, time.Since(t0), errors.New("artifactory integration not found")
		}
		integration := jobCtx.Integrations.ArtifactManager

		repository := integration.Get(sdk.ArtifactoryConfigRepositoryPrefix) + "-docker"
		rtURLRaw := integration.Get(sdk.ArtifactoryConfigURL)
		if !strings.HasSuffix(rtURLRaw, "/") {
			rtURLRaw = rtURLRaw + "/"
		}
		rtURL, err := url.Parse(rtURLRaw)
		if err != nil {
			return nil, time.Since(t0), err
		}

		destination = repository + "." + rtURL.Host + "/" + img.Repository + ":" + tag

		result.Detail.Data = sdk.V2WorkflowRunResultDockerDetail{
			Name:         destination,
			ID:           img.ImageID,
			HumanSize:    img.Size,
			HumanCreated: img.Created,
		}

		// Check if we need to tag the image
		if destination != img.Repository+":"+img.Tag {
			if err := cli.ImageTag(ctx, img.ImageID, destination); err != nil {
				return nil, time.Since(t0), errors.Errorf("unable to tag %q to %q: %v", source, destination, err)
			}
		}

		auth := registry.AuthConfig{
			Username:      integration.Get(sdk.ArtifactoryConfigTokenName),
			Password:      integration.Get(sdk.ArtifactoryConfigToken),
			ServerAddress: repository + "." + rtURL.Host,
		}
		buf, _ := json.Marshal(auth)
		registryAuth = base64.URLEncoding.EncodeToString(buf)

		output, err := cli.ImagePush(ctx, destination, types.ImagePushOptions{RegistryAuth: registryAuth})
		if err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to push %q: %v", destination, err)
		}

		if err := jsonmessage.DisplayJSONMessagesToStream(output, streams.NewOut(os.Stdout), nil); err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to push %q: %v", destination, err)
		}

		var rtConfig = grpcplugins.ArtifactoryConfig{
			URL:   rtURL.String(),
			Token: integration.Get(sdk.ArtifactoryConfigToken),
		}

		maturity := integration.Get(sdk.ArtifactoryConfigPromotionLowMaturity)
		rtFolderPath := img.Repository + "/" + tag

		// here, we GetArtifactoryFolderInfo from the repository+"-"+maturity (=generaly ...-docker-snaphot repo)
		// if a docker image exists on a remote repo, with the same name on local repo, then we want to getFileInfo from the layers pushed only.
		// Example:
		// a multi-arch exists on the remote repo with a list.manifest.json
		// docker push is done on virtual with the same name:tag, pushing manifest.json and not list.manifest.json
		// the rtFolderPathInfo should not include the list.manifest.json
		rtFolderPathInfo, err := grpcplugins.GetArtifactoryFolderInfo(ctx, &actPlugin.Common, rtConfig, repository+"-"+maturity, rtFolderPath)
		if err != nil {
			return nil, time.Since(t0), err
		}

		var manifestFound bool
		for _, child := range rtFolderPathInfo.Children {
			if strings.HasSuffix(child.URI, "manifest.json") { // Can be manifest.json of list.manifest.json for multi-arch docker image
				rtPathInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &actPlugin.Common, rtConfig, repository+"-"+maturity, rtFolderPath+child.URI)
				if err != nil {
					return nil, time.Since(t0), err
				}
				manifestFound = true
				localRepo := fmt.Sprintf("%s-%s", repository, integration.Get(sdk.ArtifactoryConfigPromotionLowMaturity))

				grpcplugins.ExtractFileInfoIntoRunResult(result, *rtPathInfo, destination, "docker", localRepo, repository, maturity)
				result.ArtifactManagerMetadata.Set("id", img.ImageID)
				break
			}
		}
		result.ArtifactManagerMetadata.Set("dir", rtFolderPathInfo.Path)
		if !manifestFound {
			return nil, time.Since(t0), errors.New("unable to get uploaded image manifest")
		}

	default:
		// Push on the registry set as parameter
		if registryURL == "" && registryAuth == "" {
			return nil, time.Since(t0), errors.New("wrong usage: <registry> and <registryAuth> parameters should not be both empty")
		}

		destination = img.Repository + ":" + tag
		if registryURL != "" {
			destination = registryURL + "/" + destination
		}

		if tag != img.Tag { // if the image already has the right tag, nothing to do
			if err := cli.ImageTag(ctx, img.ImageID, destination); err != nil {
				return nil, time.Since(t0), errors.Errorf("unable to tag %q to %q: %v", source, destination, err)
			}
		}

		output, err := cli.ImagePush(ctx, destination, types.ImagePushOptions{RegistryAuth: registryAuth})
		if err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to push %q: %v", destination, err)
		}

		if err := jsonmessage.DisplayJSONMessagesToStream(output, streams.NewOut(os.Stdout), nil); err != nil {
			return nil, time.Since(t0), errors.Errorf("unable to push %q: %v", destination, err)
		}

		result.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
		result.ArtifactManagerMetadata.Set("registry", registryURL)
		result.ArtifactManagerMetadata.Set("name", destination)
		result.ArtifactManagerMetadata.Set("id", img.ImageID)
	}

	details, err := sdk.GetConcreteDetail[sdk.V2WorkflowRunResultDockerDetail](result)
	if err != nil {
		return nil, time.Since(t0), err
	}
	details.Name = destination
	result.Detail.Data = details
	result.Status = sdk.V2WorkflowRunResultStatusCompleted

	updatedRunresult, err := grpcplugins.UpdateRunResult(ctx, &actPlugin.Common, &workerruntime.V2RunResultRequest{RunResult: result})
	return updatedRunresult.RunResult, time.Since(t0), err

}

func HumanDuration(seconds int64) string {
	createdAt := time.Unix(seconds, 0)

	if createdAt.IsZero() {
		return ""
	}
	// https://github.com/docker/cli/blob/0e70f1b7b831565336006298b9443b015c3c87a5/cli/command/formatter/buildcache.go#L156
	return units.HumanDuration(time.Now().UTC().Sub(createdAt)) + " ago"
}

func HumanSize(size int64) string {
	// https://github.com/docker/cli/blob/0e70f1b7b831565336006298b9443b015c3c87a5/cli/command/formatter/buildcache.go#L148
	return units.HumanSizeWithPrecision(float64(size), 3)
}
