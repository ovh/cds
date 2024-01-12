package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/fsamin/go-repo"
	"github.com/golang/protobuf/ptypes/empty"

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

func (actPlugin *checkoutPlugin) Run(ctx context.Context, q *actionplugin.ActionQuery) (*actionplugin.ActionResult, error) {
	gitURL := q.GetOptions()["git-url"]
	ref := q.GetOptions()["ref"]
	sha := q.GetOptions()["sha"]
	sshKey := q.GetOptions()["ssh-key"]
	path := q.GetOptions()["path"]
	authUsername := q.GetOptions()["username"]
	authToken := q.GetOptions()["token"]
	submodules := q.GetOptions()["submodules"]

	var authOption repo.Option
	if sshKey == "" {
		authOption = repo.WithHTTPAuth(authUsername, authToken)
	} else {
		authOption = repo.WithSSHAuth([]byte(sshKey))
	}

	fmt.Printf("Start cloning %s\n", gitURL)

	clonedRepo, err := repo.Clone(ctx, path, gitURL, authOption)
	if err != nil {
		return nil, fmt.Errorf("unable to clone the repository %s: %v", gitURL, err)
	}

	switch {
	case strings.HasPrefix(ref, sdk.GitRefTagPrefix):
		tag := strings.TrimPrefix(ref, sdk.GitRefTagPrefix)
		fmt.Printf("Checkout tag %s\n", tag)
		if err := clonedRepo.FetchRemoteTag(ctx, "origin", tag); err != nil {
			return nil, fmt.Errorf("unable to get tag %s: %v", tag, err)
		}
	default:
		branch := strings.TrimPrefix(ref, sdk.GitRefBranchPrefix)
		fmt.Printf("Checkout branch %s\n", branch)
		if err := clonedRepo.Checkout(ctx, branch); err != nil {
			return nil, fmt.Errorf("unable to git checkout on branch %s: %v", branch, err)
		}

		// Check commit
		if sha != "" {
			currentCommit, err := clonedRepo.LatestCommit(ctx)
			if err != nil {
				return nil, fmt.Errorf("unable to get current commit: %v", err)
			}
			if currentCommit.LongHash != sha {
				// Not the same commit, reset HARD the commit
				fmt.Printf("Reset to commit %s\n", sha)
				if err := clonedRepo.ResetHard(ctx, sha); err != nil {
					return nil, fmt.Errorf("unable to reset hard commit %s: %v", sha, err)
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
		fmt.Printf("Start updating submodules\n")
		if err := clonedRepo.SubmoduleUpdate(ctx, subMod); err != nil {
			return nil, fmt.Errorf("unable to update submodule: %v", err)
		}
	}

	fmt.Printf("Checkout completed\n")
	return &actionplugin.ActionResult{
		Status: sdk.StatusSuccess,
	}, nil
}

func main() {
	actPlugin := checkoutPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
	return
}
