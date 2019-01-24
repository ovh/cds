package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk/grpcplugin/integrationplugin"
)

type ExamplePlugin struct {
	integrationplugin.Common
}

func (e *ExamplePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*integrationplugin.IntegrationPluginManifest, error) {
	return &integrationplugin.IntegrationPluginManifest{
		Name:        "Example Plugin",
		Author:      "François Samin",
		Description: "This is an example plugin",
		Version:     sdk.VERSION,
	}, nil
}

func (e *ExamplePlugin) Deploy(ctx context.Context, q *integrationplugin.DeployQuery) (*integrationplugin.DeployResult, error) {
	fmt.Println("YOLO !!!!")
	return &integrationplugin.DeployResult{
		Details: "none",
		Status:  "success",
	}, nil
}

func (e *ExamplePlugin) DeployStatus(ctx context.Context, q *integrationplugin.DeployStatusQuery) (*integrationplugin.DeployResult, error) {
	return &integrationplugin.DeployResult{
		Details: "none",
		Status:  "success",
	}, nil
}

func main() {
	if os.Args[1:][0] == "serve" {
		e := ExamplePlugin{}
		if err := integrationplugin.Start(context.Background(), &e); err != nil {
			panic(err)
		}
		return
	}

	//Server Part - BEGIN
	var e *ExamplePlugin
	go func() {
		e = &ExamplePlugin{}
		if err := integrationplugin.Start(context.Background(), e); err != nil {
			panic(err)
		}
	}()
	//Server Part - END

	time.Sleep(100 * time.Millisecond)

	//Client Part - BEGIN
	c, err := integrationplugin.Client(context.Background(), e.Socket)
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
