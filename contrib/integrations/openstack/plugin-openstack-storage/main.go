package main

import (
	"context"
	"fmt"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/ncw/swift"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/storageplugin"
)

type openstackStoragePlugin struct {
	storageplugin.Common
}

const containerPrefix = "cds"

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

	sw := swift.Connection{
		AuthUrl:  q.GetOptions()["cds.integration.address"],
		Region:   q.GetOptions()["cds.integration.region"],
		Tenant:   q.GetOptions()["cds.integration.tenant"],
		Domain:   q.GetOptions()["cds.integration.domain"],
		UserName: q.GetOptions()["cds.integration.user"],
		ApiKey:   q.GetOptions()["cds.integration.password"],
	}
	if err := sw.Authenticate(); err != nil {
		return fail("Unable to authenticate - region:%s username:%s err:%v", q.GetOptions()["cds.integration.region"], q.GetOptions()["cds.integration.user"], err)
	}

	container := containerPrefix + o.GetPath()
	object := o.GetName()
	//TODO escape(container, object)
	fmt.Printf("Storing /%s/%s\n", container, object)
	fmt.Printf("creating container %s\n", container)
	if err := sw.ContainerCreate(container, nil); err != nil {
		return fail("Unable to create container %s", container)
	}

	fmt.Printf("creating object %s/%s\n", container, object)

	file, errC := sw.ObjectCreate(container, object, false, "", "application/octet-stream", nil)
	if errC != nil {
		return fail("Unable to create object %s - err: %v", object,errC)
	}

	fmt.Printf("copy object %s/%s", container, object)
	if _, err := io.Copy(file, data); err != nil {
		_ = file.Close()
		_ = data.Close()
		return fail("Unable to copy object buffer %s - err: %v", object,err)
	}

	if err := file.Close(); err != nil {
		return fail("Unable to close object buffer %s - err: %v", object,err)
	}

	if err := data.Close(); err != nil {
		return fail("Unable to close data buffer: %v", err)
	}

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
