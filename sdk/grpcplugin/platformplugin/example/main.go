package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk/grpcplugin/platformplugin"
)

type ExamplePlugin struct {
	platformplugin.Common
}

func (e *ExamplePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*platformplugin.PlatformPluginManifest, error) {
	return &platformplugin.PlatformPluginManifest{
		Name:        "Example Plugin",
		Author:      "François Samin",
		Description: "This is an example plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *ExamplePlugin) Deploy(ctx context.Context, q *platformplugin.DeployQuery) (*platformplugin.DeployResult, error) {
	return &platformplugin.DeployResult{
		Details: "none",
		Status:  "success",
	}, nil
}

func (e *ExamplePlugin) DeployStatus(ctx context.Context, q *platformplugin.DeployStatusQuery) (*platformplugin.DeployResult, error) {
	return &platformplugin.DeployResult{
		Details: "none",
		Status:  "success",
	}, nil
}

func main() {
	if os.Args[1:][0] == "serve" {
		e := ExamplePlugin{}
		if err := platformplugin.Start(context.Background(), &e); err != nil {
			panic(err)
		}
		return
	}

	//Server Part - BEGIN
	var e *ExamplePlugin
	go func() {
		e = &ExamplePlugin{}
		if err := platformplugin.Start(context.Background(), e); err != nil {
			panic(err)
		}
	}()
	//Server Part - END

	time.Sleep(100 * time.Millisecond)

	//Client Part - BEGIN
	c, err := platformplugin.Client(context.Background(), e.Socket)
	if err != nil {
		panic(err)
	}

	manifest, err := c.Manifest(context.Background(), new(empty.Empty))
	if err != nil {
		panic(err)
	}

	fmt.Println(manifest)
	//Client part - END
}
