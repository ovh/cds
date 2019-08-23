package action

import (
	"context"
	"fmt"
	"regexp"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs/git"
)

func RunCheckoutApplication(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	// Load action param
	directory := sdk.ParameterFind(a.Parameters, "directory")

	// Load build param
	branch := sdk.ParameterFind(params, "git.branch")
	defaultBranch := sdk.ParameterValue(params, "git.default_branch")
	tag := sdk.ParameterValue(params, "git.tag")
	commit := sdk.ParameterFind(params, "git.hash")

	gitURL, auth, err := vcsStrategy(params, secrets)
	if err != nil {
		return sdk.Result{}, err
	}

	//Prepare all options - clone options
	var opts = &git.CloneOpts{
		Recursive:               true,
		NoStrictHostKeyChecking: true,
		Depth:                   50,
		Tag:                     tag,
	}
	if branch != nil {
		opts.Branch = branch.Value
	} else {
		opts.SingleBranch = true
	}

	// if there is no branch, check if there a defaultBranch
	if (opts.Branch == "" || opts.Branch == "{{.git.branch}}") && defaultBranch != "" && tag == "" {
		opts.Branch = defaultBranch
		opts.SingleBranch = false
		wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
	}

	r := regexp.MustCompile("{{.*}}")
	if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
		opts.CheckoutCommit = commit.Value
	}

	var dir string
	if directory != nil {
		dir = directory.Value
	}

	return gitClone(ctx, wk, params, gitURL, dir, auth, opts)
}
