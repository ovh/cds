package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func RunGitClone(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	url := sdk.ParameterFind(a.Parameters, "url")
	privateKey := sdk.ParameterFind(a.Parameters, "privateKey")
	user := sdk.ParameterFind(a.Parameters, "user")
	password := sdk.ParameterFind(a.Parameters, "password")
	branch := sdk.ParameterFind(a.Parameters, "branch")
	defaultBranch := sdk.ParameterValue(params, "git.default_branch")
	tag := sdk.ParameterValue(a.Parameters, "tag")
	commit := sdk.ParameterFind(a.Parameters, "commit")
	directory := sdk.ParameterFind(a.Parameters, "directory")
	depth := sdk.ParameterFind(a.Parameters, "depth")
	submodules := sdk.ParameterFind(a.Parameters, "submodules")

	deprecatedKey := true

	if privateKey != nil && (strings.HasPrefix(privateKey.Value, "app-") || strings.HasPrefix(privateKey.Value, "proj-") || strings.HasPrefix(privateKey.Value, "env-")) {
		deprecatedKey = false
	}
	var key *vcs.SSHKey
	if privateKey != nil {
		// TODO find the key from the deprecated way or not
		if deprecatedKey {

		} else {

		}
	}

	//Prepare all options - credentials
	var auth *git.AuthOpts
	if user != nil || password != nil {
		auth = new(git.AuthOpts)
		if user != nil {
			auth.Username = user.Value
		}
		if password != nil {
			auth.Password = password.Value
		}
	}

	if key != nil {
		if auth == nil {
			auth = new(git.AuthOpts)
		}
		auth.PrivateKey = *key
	}

	var gitURL string
	if url != nil {
		gitURL = url.Value
	}

	// if not auth setted in GitClone action, we try to use VCS Strategy
	// if failed with VCS Strategy: warn only user (user can use GitClone without auth, with a git url valid)
	if gitURL == "" && (auth == nil || (auth.Username == "" && auth.Password == "" && len(auth.PrivateKey.Content) == 0)) {
		wk.SendLog(workerruntime.LevelInfo, "no url and auth parameters, trying to use VCS Strategy from application")
		var err error
		gitURL, auth, err = vcsStrategy(params, secrets)
		if err != nil {
			return sdk.Result{}, fmt.Errorf("Could not use VCS Auth Strategy from application: %v", err)
		}
	}

	if gitURL == "" {
		return sdk.Result{}, fmt.Errorf("Git repository URL is not set. Nothing to perform")
	}

	//If url is not http(s), a key must be found
	if !strings.HasPrefix(gitURL, "http") {
		if key == nil {
			return sdk.Result{}, errors.New("SSH Key not found. Unable to perform git clone")
		}
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
	if depth != nil {
		if depth.Value == "false" {
			opts.Depth = 0
		} else if depth.Value != "" {
			depthVal, errConv := strconv.Atoi(depth.Value)
			if errConv != nil {
				return sdk.Result{}, fmt.Errorf("invalid depth value. It must by empty, or false, or a numeric value. current value: %s", depth.Value)
			}
			opts.Depth = depthVal
		}
	}
	if submodules != nil && submodules.Value == "false" {
		opts.Recursive = false
	}

	// if there is no branch, check if there a defaultBranch
	if (opts.Branch == "" || opts.Branch == "{{.git.branch}}") && defaultBranch != "" && tag == "" {
		opts.Branch = defaultBranch
		opts.SingleBranch = false
		wk.SendLog(workerruntime.LevelInfo, fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
	}

	r, _ := regexp.Compile("{{.*}}")
	if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
		opts.CheckoutCommit = commit.Value
	}

	var dir string
	if directory != nil {
		dir = directory.Value
	}
	return gitClone(ctx, wk, params, gitURL, dir, auth, opts)
}

func gitClone(ctx context.Context, w workerruntime.Runtime, params []sdk.Parameter, url string, dir string, auth *git.AuthOpts, clone *git.CloneOpts) (sdk.Result, error) {
	// Install ssh key

	//Prepare all options - logs
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &git.OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	git.LogFunc = log.Info
	//Perform the git clone
	userLogCommand, err := git.Clone(url, dir, auth, clone, output)

	w.SendLog(workerruntime.LevelInfo, userLogCommand)

	//Send the logs
	if len(stdOut.Bytes()) > 0 {
		w.SendLog(workerruntime.LevelInfo, stdOut.String())
	}
	if len(stdErr.Bytes()) > 0 {
		w.SendLog(workerruntime.LevelWarn, stdErr.String())
	}

	if err != nil {
		return sdk.Result{}, fmt.Errorf("Unable to git clone: %s", err)
	}

	// extract info only if we git clone the same repo as current application linked to the pipeline
	gitURLSSH := sdk.ParameterValue(params, "git.url")
	gitURLHTTP := sdk.ParameterValue(params, "git.http_url")
	var vars []sdk.Variable
	if gitURLSSH == url || gitURLHTTP == url {
		vars, _ = extractInfo(ctx, w, dir, params, clone.Tag, clone.Branch, clone.CheckoutCommit)
	}

	stdTaglistErr := new(bytes.Buffer)
	stdTagListOut := new(bytes.Buffer)
	outputGitTag := &git.OutputOpts{
		Stderr: stdTaglistErr,
		Stdout: stdTagListOut,
	}

	errTag := git.TagList(url, dir, auth, outputGitTag)

	if len(stdTaglistErr.Bytes()) > 0 {
		w.SendLog(workerruntime.LevelInfo, stdTaglistErr.String())
	}

	if errTag != nil {
		return sdk.Result{}, fmt.Errorf("Unable to list tag for getting current version: %s", errTag)
	}

	return sdk.Result{Status: sdk.StatusSuccess, NewVariables: vars}, nil
}

func extractInfo(ctx context.Context, w workerruntime.Runtime, dir string, params []sdk.Parameter, tag, branch, commit string) ([]sdk.Variable, error) {
	var res []sdk.Variable
	author := sdk.ParameterValue(params, "git.author")
	authorEmail := sdk.ParameterValue(params, "git.author.email")
	message := sdk.ParameterValue(params, "git.message")

	info := git.ExtractInfo(dir)

	cdsVersion := sdk.ParameterFind(params, "cds.version")
	if cdsVersion == nil || cdsVersion.Value == "" {
		return nil, fmt.Errorf("cds.version is empty")
	}

	var cdsSemver string
	if info.GitDescribe != "" {
		gitDescribe := sdk.Variable{
			Name:  "git.describe",
			Type:  sdk.StringVariable,
			Value: info.GitDescribe,
		}
		res = append(res, gitDescribe)

		smver, err := semver.ParseTolerant(info.GitDescribe)
		if err != nil {
			w.SendLog(workerruntime.LevelError, fmt.Sprintf("!! WARNING !! git describe %s is not semver compatible, we can't create cds.semver variable", info.GitDescribe))
		} else {
			// Prerelease versions
			// for 0.31.1-4-g595de235a, smver.Pre = 4-g595de235a
			if len(smver.Pre) == 1 {
				tuple := strings.Split(smver.Pre[0].String(), "-")
				// we split 4-g595de235a, g595de235a is the sha1
				if len(tuple) == 2 {
					cdsSemver = fmt.Sprintf("%d.%d.%d-%s+sha.%s.cds.%s",
						smver.Major,
						smver.Minor,
						smver.Patch,
						tuple[0],
						tuple[1],
						cdsVersion.Value,
					)
				}
			}
		}

		if cdsSemver == "" {
			// here, there is no prerelease version, it's a tag
			cdsSemver = fmt.Sprintf("%d.%d.%d+cds.%s",
				smver.Major,
				smver.Minor,
				smver.Patch,
				cdsVersion.Value,
			)
		}

		// if git.describe contains a prefix 'v', we keep it
		if strings.HasPrefix(info.GitDescribe, "v") {
			cdsSemver = fmt.Sprintf("v%s", cdsSemver)
		}

	} else {
		// default value if there is no tag on repository
		cdsSemver = fmt.Sprintf("0.0.1+cds.%s", cdsVersion.Value)
	}

	if cdsSemver != "" {
		semverVar := sdk.Variable{
			Name:  "cds.semver",
			Type:  sdk.StringVariable,
			Value: cdsSemver,
		}
		res = append(res, semverVar)
	}

	if tag != "" && tag != sdk.DefaultGitCloneParameterTagValue {
		w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.tag: %s", tag))
	} else {
		if branch == "" || branch == "{{.git.branch}}" {
			if info.Branch != "" {
				gitBranch := sdk.Variable{
					Name:  "git.branch",
					Type:  sdk.StringVariable,
					Value: info.Branch,
				}
				res = append(res, gitBranch)

				w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.branch: %s", info.Branch))
			} else {
				w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.branch: [empty]"))
			}
		} else if branch != "" && branch != "{{.git.branch}}" {
			w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.branch: %s", branch))
		}

		if commit == "" || commit == "{{.git.hash}}" {
			if info.Hash != "" {
				res = append(res, sdk.Variable{
					Name:  "git.hash",
					Type:  sdk.StringVariable,
					Value: info.Hash,
				})

				hashShort := info.Hash
				if len(hashShort) >= 7 {
					hashShort = hashShort[:7]
				}
				res = append(res, sdk.Variable{
					Name:  "git.hash.short",
					Type:  sdk.StringVariable,
					Value: hashShort,
				})

				w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.hash: %s", info.Hash))
			} else {
				w.SendLog(workerruntime.LevelInfo, "git.hash: [empty]")
			}
		} else {
			w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.hash: %s", commit))
		}
	}

	if message == "" {
		if info.Message != "" {
			gitMessage := sdk.Variable{
				Name:  "git.message",
				Type:  sdk.StringVariable,
				Value: info.Message,
			}
			res = append(res, gitMessage)

			w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.message: %s", info.Message))
		} else {
			w.SendLog(workerruntime.LevelInfo, "git.message: [empty]")
		}
	} else {
		w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.message: %s", message))
	}

	if author == "" {
		if info.Author != "" {
			gitAuthor := sdk.Variable{
				Name:  "git.author",
				Type:  sdk.StringVariable,
				Value: info.Author,
			}

			res = append(res, gitAuthor)
			w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.author: %s", info.Author))
		} else {
			w.SendLog(workerruntime.LevelInfo, "git.author: [empty]")
		}
	} else {
		w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.author: %s", author))
	}

	if authorEmail == "" {
		if info.AuthorEmail != "" {
			gitAuthorEmail := sdk.Variable{
				Name:  "git.author.email",
				Type:  sdk.StringVariable,
				Value: info.AuthorEmail,
			}

			res = append(res, gitAuthorEmail)
			w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.author.email: %s", info.AuthorEmail))
		} else {
			w.SendLog(workerruntime.LevelInfo, "git.author.email: [empty]")
		}
	} else {
		w.SendLog(workerruntime.LevelInfo, fmt.Sprintf("git.author.email: %s", authorEmail))
	}
	return res, nil
}
