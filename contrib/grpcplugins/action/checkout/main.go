package main

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/user"
	"strings"

	"github.com/fsamin/go-repo"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/ovh/cds/contrib/grpcplugins"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type checkoutPlugin struct {
	actionplugin.Common
}

func (actPlugin *checkoutPlugin) Manifest(_ context.Context, _ *empty.Empty) (*actionplugin.ActionPluginManifest, error) {
	return &actionplugin.ActionPluginManifest{
		Name:        "plugin-checkout",
		Author:      "Steven GUIHEUX <steven.guiheux@ovhcloud.com>",
		Description: `This action checkout a git repository`,
		Version:     sdk.VERSION,
	}, nil
}

func (p *checkoutPlugin) Stream(q *actionplugin.ActionQuery, stream actionplugin.ActionPlugin_StreamServer) error {
	ctx := context.Background()
	p.StreamServer = stream

	gitURL := q.GetOptions()["git-url"]
	ref := q.GetOptions()["ref"]
	sha := q.GetOptions()["sha"]
	sshKey := q.GetOptions()["ssh-key"]
	path := q.GetOptions()["path"]
	authUsername := q.GetOptions()["username"]
	authToken := q.GetOptions()["token"]
	submodules := q.GetOptions()["submodules"]

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	var key *sdk.ProjectKey
	var authOption repo.Option
	if sshKey == "" {
		authOption = repo.WithHTTPAuth(authUsername, authToken)
	} else {
		var err error
		key, err = grpcplugins.GetProjectKey(ctx, &p.Common, sshKey)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to retrieve sshkey %s: %v", sshKey, err)
			return stream.Send(res)
		}
		authOption = repo.WithSSHAuth([]byte(key.Private))
	}

	// Create directory
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.FileMode(0755)); err != nil {
			return err
		}
	}

	grpcplugins.Logf(&p.Common, "Start cloning %s\n", gitURL)

	clonedRepo, err := repo.Clone(ctx, path, gitURL, authOption)
	if err != nil {
		res.Status = sdk.StatusFail
		res.Details = fmt.Sprintf("unable to clone the repository %s: %v", gitURL, err)
		return stream.Send(res)
	}

	switch {
	case strings.HasPrefix(ref, sdk.GitRefTagPrefix):
		tag := strings.TrimPrefix(ref, sdk.GitRefTagPrefix)
		grpcplugins.Logf(&p.Common, "Checkout tag %s\n", tag)
		if err := clonedRepo.FetchRemoteTag(ctx, "origin", tag); err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to get tag %s: %v", tag, err)
			return stream.Send(res)
		}
	default:
		branch := strings.TrimPrefix(ref, sdk.GitRefBranchPrefix)
		grpcplugins.Logf(&p.Common, "Checkout branch %s\n", branch)
		if err := clonedRepo.Checkout(ctx, branch); err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to git checkout on branch %s: %v", branch, err)
			return stream.Send(res)
		}

		// Check commit
		if sha != "" && sha != "HEAD" {
			currentCommit, err := clonedRepo.LatestCommit(ctx, repo.CommitOption{DisableDiffDetail: true})
			if err != nil {
				res.Status = sdk.StatusFail
				res.Details = fmt.Sprintf("unable to get current commit: %v", err)
				return stream.Send(res)
			}
			if currentCommit.LongHash != sha {
				// Not the same commit, reset HARD the commit
				grpcplugins.Logf(&p.Common, "Reset to commit %s\n", sha)
				if err := clonedRepo.ResetHard(ctx, sha); err != nil {
					res.Status = sdk.StatusFail
					res.Details = fmt.Sprintf("unable to reset hard commit %s: %v", sha, err)
					return stream.Send(res)
				}
			}
		}

	}

	if submodules == "true" || submodules == "recursive" {
		subMod := repo.SubmoduleOpt{
			Init: true,
		}
		if submodules == "recursive" {
			subMod.Recursive = true
		}
		grpcplugins.Logf(&p.Common, "Start updating submodules\n")
		if err := clonedRepo.SubmoduleUpdate(ctx, subMod); err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to update submodule: %v", err)
			return stream.Send(res)
		}
	}
	grpcplugins.Logf(&p.Common, "Checkout completed\n")

	if key != nil {
		// Install key
		workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
		if err != nil {
			err := fmt.Errorf("unable to get working directory: %v", err)
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
		u, err := user.Current()
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to get current user: %v", err)
			return stream.Send(res)
		}
		// Install id_rsa priv key
		if u != nil && u.HomeDir != "" {
			sshFilePath := u.HomeDir + "/.ssh/id_rsa"
			if _, err := grpcplugins.InstallSSHKey(ctx, &p.Common, workDirs, sshKey, sshFilePath, key.Private); err != nil {
				err := fmt.Errorf("unable to install sshkey on worker: %v", err)
				res.Status = sdk.StatusFail
				res.Details = err.Error()
				return stream.Send(res)
			}

			urlParsed, err := url.Parse(gitURL)
			if err != nil {
				return fmt.Errorf("unable to parse git url: %s", gitURL)
			}
			host, port, _ := net.SplitHostPort(urlParsed.Host)
			if port == "" {
				port = "22"
			}

			goRoutines := sdk.NewGoRoutines(ctx)
			workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
			if err != nil {
				return fmt.Errorf("unable to get working directory: %v", err)
			}

			scriptContent := fmt.Sprintf("ssh-keyscan -t rsa -p %s %s >> %s/.ssh/known_hosts", port, host, u.HomeDir)

			chanRes := make(chan *actionplugin.ActionResult)
			goRoutines.Exec(ctx, "InstallSSHKey-runScript", func(ctx context.Context) {
				grpcplugins.RunScript(ctx, &p.Common, chanRes, workDirs.WorkingDir, scriptContent)
			})

			select {
			case <-ctx.Done():
				return fmt.Errorf("CDS Worker execution canceled: " + ctx.Err().Error())
			case result := <-chanRes:
				if result.Status == sdk.StatusFail {
					return stream.Send(res)
				}
			}
		}
	}

	// Install and import GPG Key
	gpgkey, err := grpcplugins.GetProjectKey(ctx, &p.Common, gpgKeyfromIntegration)
	if err != nil {

	}
	if _, _, err := sdk.ImportGPGKey(workDirs.BaseDir, gpgkey.Name, gpgkey.Private); err != nil {
		return fmt.Errorf("unable to install pgp key %s: %v", gpgkey, err)
	}

	return stream.Send(res)

}

func (actPlugin *checkoutPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func main() {
	actPlugin := checkoutPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
