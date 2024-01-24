package main

import (
	"context"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/go-units"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/moby/moby/client"
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

	tagSlice := strings.Split(tags, ",")

	if err := actPlugin.perform(ctx, image, tagSlice, registry); err != nil {
		res.Status = sdk.StatusFail
		res.Status = err.Error()
		return res, err
	}

	return res, nil
}

func (actPlugin *dockerPushPlugin) perform(ctx context.Context, image string, tags []string, registry string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return sdk.Errorf("unable to get instanciate docker client: %v", err)
	}

	imageSummaries, err := cli.ImageList(ctx, types.ImageListOptions{All: true})
	if err != nil {
		return sdk.Errorf("unable to get docker image %q: %v", image, err)
	}

	type img struct {
		repository string
		tag        string
		imageID    string
		created    string
		size       string
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

	for _, tag := range tags {
		var destination = registry + "/" + imgFound.repository + ":" + tag
		if err := cli.ImageTag(ctx, imgFound.imageID, destination); err != nil {
			return sdk.Errorf("unable to tag %q: %v", image, err)
		}

		output, err := cli.ImagePush(ctx, destination, types.ImagePushOptions{})
		if err != nil {
			return sdk.Errorf("unable to push %q: %v", image, err)
		}

	}

	return nil
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
