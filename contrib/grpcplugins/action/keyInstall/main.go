package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/spf13/afero"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/ovh/cds/sdk/vcs"
)

type keyInstallPlugin struct {
	actionplugin.Common
}

func (actPlugin *keyInstallPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "keyInstall",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action install a ssh or gpg key`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *keyInstallPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	keyName := q.GetOptions()["keyName"]
	path := q.GetOptions()["filePath"]

	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	if err := p.perform(ctx, workDirs, keyName, path); err != nil {
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	return stream.Send(res)

}

func (actPlugin *keyInstallPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func (actPlugin *keyInstallPlugin) perform(ctx context.Context, workDirs *sdk.WorkerDirectories, keyName, filePath string) error {
	key, err := grpcplugins.GetProjectKey(ctx, &actPlugin.Common, keyName)
	if err != nil {
		return err
	}

	switch key.Type {
	case sdk.KeyTypeSSH:
		if filePath == "" {
			filePath = ".ssh/id_rsa-" + keyName
		}
		if sdk.PathIsAbs(filePath) {
			return fmt.Errorf("unable to use an absolute path: %s", filePath)
		}

		absPath, err := filepath.Abs(filepath.Join(workDirs.WorkingDir, filePath))
		if err != nil {
			return fmt.Errorf("unable to compute ssh key absolute path: %v", err)
		}

		destinationDirectory := filepath.Dir(absPath)
		if err := afero.NewOsFs().MkdirAll(destinationDirectory, os.FileMode(0755)); err != nil {
			return fmt.Errorf("unable to create directory %s: %v", destinationDirectory, err)
		}

		if err := vcs.WriteKey(afero.NewOsFs(), absPath, key.Private); err != nil {
			return fmt.Errorf("cannot setup ssh key %s : %v", key.Name, err)
		}
		return nil
	case sdk.KeyTypePGP:
		if _, _, err := sdk.ImportGPGKey("", key.Name, key.Private); err != nil {
			return fmt.Errorf("unable to install pgp key %s: %v", keyName, err)
		}
		return nil
	default:
		return fmt.Errorf("unknown key type [%s]", key.Type)
	}
}

func main() {
	actPlugin := keyInstallPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
