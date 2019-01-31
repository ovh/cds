package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/storageplugin"
)

// ExamplePlugin represents an example plugins
type ExamplePlugin struct {
	storageplugin.Common
}

// Manifest implementation
func (e *ExamplePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*storageplugin.StoragePluginManifest, error) {
	return &storageplugin.StoragePluginManifest{
		Name:        "Example Plugin",
		Author:      "Yvonnick Esnault",
		Description: "This is an example plugin",
		Version:     sdk.VERSION,
	}, nil
}

// ArtifactUpload implementation
func (e *ExamplePlugin) ArtifactUpload(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// ArtifactDownload implementation
func (e *ExamplePlugin) ArtifactDownload(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// ServeStaticFiles implementation
func (e *ExamplePlugin) ServeStaticFiles(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// CachePull implementation
func (e *ExamplePlugin) CachePull(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// CachePush implementation
func (e *ExamplePlugin) CachePush(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

func main() {
	if os.Args[1:][0] == "serve" {
		e := ExamplePlugin{}
		if err := storageplugin.Start(context.Background(), &e); err != nil {
			panic(err)
		}
		return
	}

	//Server Part - BEGIN
	var e *ExamplePlugin
	go func() {
		e = &ExamplePlugin{}
		if err := storageplugin.Start(context.Background(), e); err != nil {
			panic(err)
		}
	}()
	//Server Part - END

	time.Sleep(100 * time.Millisecond)

	//Client Part - BEGIN
	c, err := storageplugin.Client(context.Background(), e.Socket)
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
