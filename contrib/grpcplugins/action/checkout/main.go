package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
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

	gitURL := sanitizeGitURL(q.GetOptions()["git-url"])
	ref := q.GetOptions()["ref"]
	sha := q.GetOptions()["sha"]
	sshKey := q.GetOptions()["ssh-key"]
	path := q.GetOptions()["path"]
	authUsername := q.GetOptions()["username"]
	authToken := q.GetOptions()["token"]
	submodules := q.GetOptions()["submodules"]
	gpgKey := q.GetOptions()["gpg-key"]
	email := q.GetOptions()["email"]
	depthS := q.GetOptions()["depth"]

	res := &actionplugin.StreamResult{
		Status: sdk.StatusSuccess,
	}

	var key *sdk.ProjectKey
	var gitOptions []repo.Option
	if sshKey == "" {
		gitOptions = append(gitOptions, repo.WithHTTPAuth(authUsername, authToken))
	} else {
		var err error
		key, err = grpcplugins.GetProjectKey(ctx, &p.Common, sshKey)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to retrieve sshkey %s: %v", sshKey, err)
			return stream.Send(res)
		}
		gitOptions = append(gitOptions, repo.WithSSHAuth([]byte(key.Private)))
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
			sshFilePath := filepath.Join(u.HomeDir, ".ssh", "id_rsa")
			if _, err := grpcplugins.InstallSSHKey(ctx, &p.Common, workDirs, sshKey, sshFilePath, key.Private); err != nil {
				err := fmt.Errorf("unable to install sshkey on worker: %v", err)
				res.Status = sdk.StatusFail
				res.Details = err.Error()
				return stream.Send(res)
			}
			os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", filepath.ToSlash(sshFilePath)))
		}
	}

	if email != "" {
		grpcplugins.Logf(&p.Common, "Setting up git config (user.name=%s, user.email=%s)\n", authUsername, email)
		if err := p.Exec(ctx, workDirs, fmt.Sprintf(`git config --global user.email "%s"`, email)); err != nil {
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
		if err := p.Exec(ctx, workDirs, fmt.Sprintf(`git config --global user.name "%s"`, authUsername)); err != nil {
			res.Status = sdk.StatusFail
			res.Details = err.Error()
			return stream.Send(res)
		}
	}

	// Install and import GPG Key if gpg is installed
	var gpgFound bool
	if _, err := exec.LookPath("gpg2"); err == nil {
		gpgFound = true
	}
	if !gpgFound {
		if _, err := exec.LookPath("gpg"); err == nil {
			gpgFound = true
		}
	}
	if gpgFound && gpgKey != "" {
		k, err := grpcplugins.GetProjectKey(ctx, &p.Common, gpgKey)
		if err != nil {
			res.Status = sdk.StatusFail
			res.Details = fmt.Sprintf("unable to get GPG key %q: %v", gpgKey, err)
			return stream.Send(res)
		}

		if !sdk.IsGPGKeyAlreadyInstalled(k.LongKeyID) {
			grpcplugins.Logf(&p.Common, "Installing GPG Key %s\n", gpgKey)
			if _, _, err := sdk.ImportGPGKey(workDirs.BaseDir, k.Name, []byte(k.Private)); err != nil {
				res.Status = sdk.StatusFail
				res.Details = fmt.Sprintf("unable to install GPG key %q: %v", gpgKey, err)
				return stream.Send(res)
			}
			grpcplugins.Logf(&p.Common, "Setting up git config (user.signingkey=%s)...\n", k.KeyID)
			if err := p.Exec(ctx, workDirs, fmt.Sprintf(`git config --global user.signingkey "%s"`, k.KeyID)); err != nil {
				res.Status = sdk.StatusFail
				res.Details = err.Error()
				return stream.Send(res)
			}
			if err := p.Exec(ctx, workDirs, `git config --global commit.gpgsign true`); err != nil {
				res.Status = sdk.StatusFail
				res.Details = err.Error()
				return stream.Send(res)
			}
		}

	} else if gpgKey != "" {
		grpcplugins.Logf(&p.Common, "Can't install GPG Key %q, gpg/gpg2 is not available\n", gpgKey)
	}

	if depthS != "" {
		d, err := strconv.Atoi(depthS)
		if err != nil {
			res := &actionplugin.StreamResult{
				Status:  sdk.StatusFail,
				Details: fmt.Sprintf("invalid depth value: %v", err),
			}
			return stream.Send(res)
		}
		grpcplugins.Logf(&p.Common, "Setting git clone depth to %d\n", d)
		gitOptions = append(gitOptions, repo.WithDepth(d))
	}

	grpcplugins.Logf(&p.Common, "Start cloning %s\n", gitURL)

	clonedRepo, err := repo.Clone(ctx, path, gitURL, gitOptions...)
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

// sanitizeGitURL fixes hybrid SSH URLs that mix the ssh:// scheme with the
// SCP-like host:path syntax. For example GitHub returns "git@github.com:ovh/cds.git"
// but some code paths prepend "ssh://" producing "ssh://git@github.com:ovh/cds.git".
// In the ssh:// scheme the colon is interpreted as a port separator, so ":ovh" is
// an invalid port. This function converts such URLs to the correct form:
//
//	ssh://git@github.com:ovh/cds.git  →  ssh://git@github.com/ovh/cds.git
//
// URLs that already have a valid numeric port (e.g. ssh://git@host:7999/path)
// or that don't use the ssh:// scheme are returned unchanged.
func sanitizeGitURL(raw string) string {
	if !strings.HasPrefix(raw, "ssh://") {
		return raw
	}

	u, err := url.Parse(raw)
	if err == nil && u.Port() != "" {
		// url.Parse succeeded and found a port – if it's numeric the URL is
		// already well-formed (e.g. ssh://git@host:7999/repo).
		return raw
	}

	// url.Parse failed (invalid port like ":ovh") or returned an empty port.
	// Try to detect the SCP-inside-ssh pattern: ssh://[user@]host:path
	withoutScheme := strings.TrimPrefix(raw, "ssh://")

	// Find host (possibly with user@)
	colonIdx := strings.Index(withoutScheme, ":")
	slashIdx := strings.Index(withoutScheme, "/")

	if colonIdx == -1 {
		// No colon at all – nothing to fix.
		return raw
	}

	if slashIdx != -1 && slashIdx < colonIdx {
		// Slash comes before colon (ssh://user@host/path:rest) – not the hybrid case.
		return raw
	}

	// We have ssh://something:rest where rest does NOT start with a digit
	// (otherwise url.Parse would have been fine). Replace the colon with a slash.
	return "ssh://" + withoutScheme[:colonIdx] + "/" + withoutScheme[colonIdx+1:]
}

func main() {
	actPlugin := checkoutPlugin{}
	if err := actionplugin.Start(context.Background(), &actPlugin); err != nil {
		panic(err)
	}
}
