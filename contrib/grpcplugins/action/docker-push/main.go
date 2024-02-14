package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/go-units"
	"github.com/golang/protobuf/ptypes/empty"
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
		Name:        "docker-push",
		Author:      "François SAMIN <francois.samin@corp.ovh.com>",
		Description: "Push an image docker on a docker registry",
		Version:     sdk.VERSION,
	}, nil
}

// Run implements actionplugin.ActionPluginServer.
func (actPlugin *dockerPushPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	res := &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}

	image := q.GetOptions()["image"]
	tags := q.GetOptions()["tags"]
	registry := q.GetOptions()["registry"]
	auth := q.GetOptions()["registryAuth"]

	tags = strings.Replace(tags, " ", ",", -1) // If tags are separated by <space>
	tags = strings.Replace(tags, ";", ",", -1) // If tags are separated by <semicolon>
	tagSlice := strings.Split(tags, ",")

	if err := actPlugin.perform(ctx, image, tagSlice, registry, auth); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

type img struct {
	repository string
	tag        string
	imageID    string
	created    string
	size       string
}

func (actPlugin *dockerPushPlugin) perform(ctx context.Context, image string, tags []string, registry, registryAuth string) error {
	if image == "" {
		return sdk.Errorf("wrong usage: <image> parameter should not be empty")
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return sdk.Errorf("unable to get instanciate docker client: %v", err)
	}

	imageSummaries, err := cli.ImageList(ctx, types.ImageListOptions{All: false})
	if err != nil {
		return sdk.Errorf("unable to get docker image %q: %v", image, err)
	}

	images := []img{}
	for _, image := range imageSummaries {
		repository := "<none>"
		tag := "<none>"
		if len(image.RepoTags) > 0 {
			splitted := strings.Split(image.RepoTags[0], ":")
			repository = splitted[0]
			tag = splitted[1]
		} else if len(image.RepoDigests) > 0 {
			repository = strings.Split(image.RepoDigests[0], "@")[0]
		}
		duration := HumanDuration(image.Created)
		size := HumanSize(image.Size)
		images = append(images, img{repository: repository, tag: tag, imageID: image.ID[7:19], created: duration, size: size})
	}

	var imgFound *img
	for i := range images {
		if images[i].repository+":"+images[i].tag == image {
			imgFound = &images[i]
			break
		}
	}

	if imgFound == nil {
		return sdk.Errorf("image %q not found", image)
	}

	if len(tags) == 0 { // If no tag is provided, keep the actual tag
		tags = []string{imgFound.tag}
	}

	for _, tag := range tags {
		result, d, err := actPlugin.performImage(ctx, cli, image, imgFound, registry, registryAuth, strings.TrimSpace(tag))
		if err != nil {
			grpcplugins.Error(err.Error())
			return err
		}
		grpcplugins.Logf("Image %s pushed in %.3fs", result.Name(), d.Seconds())
	}

	return nil
}

func (actPlugin *dockerPushPlugin) performImage(ctx context.Context, cli *client.Client, source string, img *img, registry string, registryAuth string, tag string) (*sdk.V2WorkflowRunResult, time.Duration, error) {
	var t0 = time.Now()

	// Create run result at status "pending"
	var runResultRequest = workerruntime.V2RunResultRequest{
		RunResult: &sdk.V2WorkflowRunResult{
			IssuedAt: time.Now(),
			Type:     sdk.V2WorkflowRunResultTypeDocker,
			Status:   sdk.V2WorkflowRunResultStatusPending,
			Detail: sdk.V2WorkflowRunResultDetail{
				Data: sdk.V2WorkflowRunResultDockerDetail{
					Name:         source,
					ID:           img.imageID,
					HumanSize:    img.size,
					HumanCreated: img.created,
				},
			},
		},
	}

	response, err := grpcplugins.CreateRunResult(ctx, &actPlugin.Common, &runResultRequest)
	if err != nil {
		return nil, time.Since(t0), err
	}

	result := response.RunResult

	var destination string
	// Upload the file to an artifactory or the docker registry
	switch {
	case result.ArtifactManagerIntegrationName != nil:
		integration, err := grpcplugins.GetIntegrationByName(ctx, &actPlugin.Common, *response.RunResult.ArtifactManagerIntegrationName)
		if err != nil {
			return nil, time.Since(t0), err
		}

		repository := integration.Config[sdk.ArtifactoryConfigRepositoryPrefix].Value + "-docker"
		rtURLRaw := integration.Config[sdk.ArtifactoryConfigURL].Value
		if !strings.HasSuffix(rtURLRaw, "/") {
			rtURLRaw = rtURLRaw + "/"
		}
		rtURL, err := url.Parse(rtURLRaw)
		if err != nil {
			return nil, time.Since(t0), err
		}

		destination = repository + "." + rtURL.Host + "/" + img.repository + ":" + tag

		result.Detail.Data = sdk.V2WorkflowRunResultDockerDetail{
			Name:         destination,
			ID:           img.imageID,
			HumanSize:    img.size,
			HumanCreated: img.created,
		}

		if tag != img.tag { // if the image already has the right tag, nothing to do
			if err := cli.ImageTag(ctx, img.imageID, destination); err != nil {
				return nil, time.Since(t0), errors.Errorf("unable to tag %q to %q: %v", source, destination, err)
			}
		}

		auth := types.AuthConfig{
			Username:      integration.Config[sdk.ArtifactoryConfigTokenName].Value,
			Password:      integration.Config[sdk.ArtifactoryConfigToken].Value,
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
			Token: integration.Config[sdk.ArtifactoryConfigToken].Value,
		}

		rtFolderPath := img.repository + "/" + tag
		rtFolderPathInfo, err := grpcplugins.GetArtifactoryFolderInfo(ctx, &actPlugin.Common, rtConfig, repository, rtFolderPath)
		if err != nil {
			return nil, time.Since(t0), err
		}

		var manifestFound bool
		for _, child := range rtFolderPathInfo.Children {
			if strings.HasSuffix(child.URI, "manifest.json") { // Can be manifest.json of list.manifest.json for multi-arch docker image
				rtPathInfo, err := grpcplugins.GetArtifactoryFileInfo(ctx, &actPlugin.Common, rtConfig, repository, rtFolderPath+child.URI)
				if err != nil {
					return nil, time.Since(t0), err
				}
				manifestFound = true
				result.ArtifactManagerMetadata = &sdk.V2WorkflowRunResultArtifactManagerMetadata{}
				result.ArtifactManagerMetadata.Set("repository", repository) // This is the virtual repository
				result.ArtifactManagerMetadata.Set("type", "docker")
				result.ArtifactManagerMetadata.Set("maturity", integration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value)
				result.ArtifactManagerMetadata.Set("name", destination)
				result.ArtifactManagerMetadata.Set("path", rtPathInfo.Path)
				result.ArtifactManagerMetadata.Set("md5", rtPathInfo.Checksums.Md5)
				result.ArtifactManagerMetadata.Set("sha1", rtPathInfo.Checksums.Sha1)
				result.ArtifactManagerMetadata.Set("sha256", rtPathInfo.Checksums.Sha256)
				result.ArtifactManagerMetadata.Set("uri", rtPathInfo.URI)
				result.ArtifactManagerMetadata.Set("mimeType", rtPathInfo.MimeType)
				result.ArtifactManagerMetadata.Set("downloadURI", rtPathInfo.DownloadURI)
				result.ArtifactManagerMetadata.Set("createdBy", rtPathInfo.CreatedBy)
				result.ArtifactManagerMetadata.Set("localRepository", repository+"-"+integration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value)
				result.ArtifactManagerMetadata.Set("id", img.imageID)
				break
			}
		}
		if !manifestFound {
			return nil, time.Since(t0), errors.New("unable to get uploaded image manifest")
		}

	default:
		// Push on the registry set as parameter
		if registry == "" && registryAuth == "" {
			return nil, time.Since(t0), errors.New("wrong usage: <registry> and <registryAuth> parameters should not be both empty")
		}

		destination = img.repository + ":" + tag
		if registry != "" {
			destination = registry + "/" + destination
		}

		if tag != img.tag { // if the image already has the right tag, nothing to do
			if err := cli.ImageTag(ctx, img.imageID, destination); err != nil {
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
		result.ArtifactManagerMetadata.Set("registry", registry)
		result.ArtifactManagerMetadata.Set("name", destination)
		result.ArtifactManagerMetadata.Set("id", img.imageID)
	}

	details, err := result.GetDetailAsV2WorkflowRunResultDockerDetail()
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
