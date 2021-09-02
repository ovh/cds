package main

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mholt/archiver/v3"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

/* Inside contrib/grpcplugins/action
$ make build archive
$ make publish archive
*/

type archiveActionPlugin struct {
	actionplugin.Common
}

func (actPlugin *archiveActionPlugin) Manifest(ctx context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-archive",
		Author:      "mdouchement <https://github.com/mdouchement>",
		Description: `This is a plugin to compress or uncompress an archive. Supported formats: .zip, .tar, .tar.gz, .tar.bz2, .tar.xz, .tar.zst, .tar.lz4, .tar.sz, and .rar (extract-only)`,
		Version:     sdk.VERSION,
	}, nil
}

func (actPlugin *archiveActionPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	source := q.GetOptions()["source"]
	destination := q.GetOptions()["destination"]
	action := q.GetOptions()["action"]

	var err error
	switch action {
	case "compress":
		err = archiver.Archive([]string{source}, destination)
	case "uncompress":
		err = archiver.Unarchive(source, destination)
	default:
		return actionplugin.Fail("Invalid action: %s", action)
	}

	if err != nil {
		return actionplugin.Fail("Could not %s: %s", action, err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := archiveActionPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
