package main

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/storageplugin"
)

type openstackStoragePlugin struct {
	storageplugin.Common
}

func (e *openstackStoragePlugin) Manifest(ctx context.Context, _ *empty.Empty) (*storageplugin.StoragePluginManifest, error) {
	return &storageplugin.StoragePluginManifest{
		Name:        "OVH Openstack Artifact Storage Plugin",
		Author:      "OVH SAS",
		Description: "OVH Openstack Artifact Storage Plugin",
		Version:     sdk.VERSION,
	}, nil
}

// ArtifactUpload implementation
func (e *openstackStoragePlugin) ArtifactUpload(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	// 	q.GetOptions()["cds.integration.address"]
	//  q.GetOptions()["cds.integration.region"]
	//  q.GetOptions()["cds.integration.domain"]
	//  q.GetOptions()["cds.integration.tenant"]
	//  q.GetOptions()["cds.integration.user"]
	//  q.GetOptions()["cds.integration.password"]
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// ArtifactDownload implementation
func (e *openstackStoragePlugin) ArtifactDownload(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// ServeStaticFiles implementation
func (e *openstackStoragePlugin) ServeStaticFiles(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// CachePull implementation
func (e *openstackStoragePlugin) CachePull(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

// CachePush implementation
func (e *openstackStoragePlugin) CachePush(ctx context.Context, q *storageplugin.Options) (*storageplugin.Result, error) {
	return &storageplugin.Result{
		Details: "none",
		Status:  "success",
	}, nil
}

func main() {
	e := openstackStoragePlugin{}
	if err := storageplugin.Start(context.Background(), &e); err != nil {
		panic(err)
	}
	return
}

func fail(format string, args ...interface{}) (*storageplugin.Result, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &storageplugin.Result{
		Details: msg,
		Status:  sdk.StatusFail.String(),
	}, nil
}
