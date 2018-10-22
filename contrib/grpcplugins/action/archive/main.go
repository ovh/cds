package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/mholt/archiver"

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
		Description: `This is a plugin to compress or uncompress an archive. Supported formats: .zip, .tar, .tar.gz, .tar.bz2, .tar.xz, .tar.lz4, .tar.sz, and .rar (extract-only)`,
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
		ff := archiver.MatchingFormat(destination)
		if ff == nil {
			return fail("Unsupported file extension for: %s", destination)
		}
		err = ff.Make(destination, []string{source})
	case "uncompress":
		ff := archiver.MatchingFormat(source)
		if ff == nil {
			return fail("Unsupported file extension for: %s", source)
		}
		err = ff.Open(source, destination)
	default:
		return fail("Invalid action: %s", action)
	}

	if err != nil {
		return fail("Could not %s: %s", action, err)
	}

	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess.String(),
	}, nil
}

func (actPlugin *archiveActionPlugin) WorkerHTTPPort(ctx context.Context, q *actionplugin.WorkerHTTPPortQuery) (*empty.Empty, error) {
	actPlugin.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func main() {
	actPlugin := archiveActionPlugin{}
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
