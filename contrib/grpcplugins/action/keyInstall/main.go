package main

import (
	"context"
	"fmt"
	"os/user"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
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
	path := q.GetOptions()["path"]

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
			u, err := user.Current()
			if err != nil {
				return fmt.Errorf("unable to get current user: %v", err)
			}
			if u != nil && u.HomeDir != "" {
				filePath = u.HomeDir + "/.ssh/id_rsa-" + keyName
			}
		}
		filePath, err := grpcplugins.InstallSSHKey(ctx, &actPlugin.Common, workDirs, keyName, filePath, key.Private)
		if err != nil {
			return err
		}
		grpcplugins.Logf(&actPlugin.Common, "To be able to use git command in a further step, you must run this command first:")
		grpcplugins.Successf(&actPlugin.Common, "export GIT_SSH_COMMAND=\"ssh -i %s -o StrictHostKeyChecking=no\"", filePath)
	case sdk.KeyTypePGP:
		if _, _, err := sdk.ImportGPGKey("", key.Name, key.Private); err != nil {
			return fmt.Errorf("unable to install pgp key %s: %v", keyName, err)
		}
		grpcplugins.Logf(&actPlugin.Common, "pgpkey %s has been imported", key.Name)
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
