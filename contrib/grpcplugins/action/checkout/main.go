package main

import (
	"context"
	"errors"
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
	gpgKey := q.GetOptions()["gpg-key"]
	email := q.GetOptions()["email"]

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
	workDirs, err := grpcplugins.GetWorkerDirectories(ctx, &p.Common)
	if err != nil {
		err := fmt.Errorf("unable to get working directory: %v", err)
		res.Status = sdk.StatusFail
		res.Details = err.Error()
		return stream.Send(res)
	}

	if key != nil {
		grpcplugins.Logf(&p.Common, "Setting up SSH Key\n")

		// Install key
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

			scriptContent := fmt.Sprintf("ssh-keyscan -t rsa -p %s %s >> %s/.ssh/known_hosts", port, host, u.HomeDir)
			if err := p.Exec(ctx, workDirs, scriptContent); err != nil {
				res.Status = sdk.StatusFail
				res.Details = err.Error()
				return stream.Send(res)
			}
		}
	}

	if email != "" {
		grpcplugins.Logf(&p.Common, "Setting up git config (user.name=%s, user.email=%s)\n", email, authUsername)
		scriptContent := fmt.Sprintf(`git config --global user.email "%s" && git config --global user.name "%s"`, email, authUsername)
		if err := p.Exec(ctx, workDirs, scriptContent); err != nil {
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
	}

	// Install and import GPG Key
	if gpgKey != "" {
		k, err := grpcplugins.GetProjectKey(ctx, &p.Common, gpgKey)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to get GPG key %q: %v", gpgKey, err)
			return stream.Send(res)
		}
		grpcplugins.Logf(&p.Common, "Installing GPG Key %s\n", gpgKey)

		if _, _, err := sdk.ImportGPGKey(workDirs.BaseDir, k.Name, k.Private); err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to install GPG key %q: %v", gpgKey, err)
			return stream.Send(res)
		}

		grpcplugins.Logf(&p.Common, "Setting up git config (user.signingkey=%s)...\n", k.KeyID)
		scriptContent := fmt.Sprintf(`git config --global user.signingkey "%s" && git config --global commit.gpgsign true`, k.KeyID)
		if err := p.Exec(ctx, workDirs, scriptContent); err != nil {
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
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

	return stream.Send(res)
}

func (p *checkoutPlugin) Exec(ctx context.Context, workDirs *sdk.WorkerDirectories, scriptContent string) error {
	goRoutines := sdk.NewGoRoutines(ctx)

	chanRes := make(chan *actionplugin.ActionResult)
	goRoutines.Exec(ctx, "checkoutPlugin-exec", func(ctx context.Context) {
		if err := grpcplugins.RunScript(ctx, &p.Common, chanRes, workDirs.WorkingDir, scriptContent); err != nil {
			chanRes <- &actionplugin.ActionResult{
				Status:  sdk.StatusFail,
				Details: err.Error(),
			}
		}
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	case result := <-chanRes:
		if result.Status == sdk.StatusFail {
			return errors.New(result.Details)
		}
	}

	return nil
}

func (actPlugin *checkoutPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	return nil, sdk.ErrNotImplemented
}

func main() {
	actPlugin := checkoutPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
