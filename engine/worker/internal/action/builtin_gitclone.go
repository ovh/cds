package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/blang/semver"
	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func RunGitClone(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	url := sdk.ParameterFind(a.Parameters, "url")
	privateKey := sdk.ParameterFind(a.Parameters, "privateKey")
	user := sdk.ParameterFind(a.Parameters, "user")
	password := sdk.ParameterFind(a.Parameters, "password")
	branch := sdk.ParameterFind(a.Parameters, "branch")
	defaultBranch := sdk.ParameterValue(wk.Parameters(), "git.default_branch")
	tag := sdk.ParameterValue(a.Parameters, "tag")
	commit := sdk.ParameterFind(a.Parameters, "commit")
	directory := sdk.ParameterFind(a.Parameters, "directory")
	depth := sdk.ParameterFind(a.Parameters, "depth")
	submodules := sdk.ParameterFind(a.Parameters, "submodules")

	var key *vcs.SSHKey
	if privateKey != nil && privateKey.Value != "" {
		// The private key parameter, contains the name of the private key to use.
		// Let's look up in the secret list to find the content of the private key
		privateKeyContent := sdk.VariableFind(secrets, "cds.key."+privateKey.Value+".priv")

		if privateKeyContent == nil {
			return sdk.Result{}, fmt.Errorf("unknown key \"%s\"", privateKey.Name)
		}

		installedKey, err := wk.InstallKey(sdk.Variable{
			Name:  privateKeyContent.Name,
			Value: privateKeyContent.Value,
			Type:  string(sdk.KeyTypeSSH),
		})
		if err != nil {
			return sdk.Result{}, err
		}
		key = &vcs.SSHKey{
			Filename: installedKey.PKey,
			Content:  installedKey.Content,
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
		wk.SendLog(ctx, workerruntime.LevelInfo, "no url and auth parameters, trying to use VCS Strategy from application")
		var err error
		gitURL, auth, err = vcsStrategy(ctx, wk, wk.Parameters(), secrets)
		if err != nil {
			return sdk.Result{}, fmt.Errorf("Could not use VCS Auth Strategy from application: %v", err)
		}
		key = &auth.PrivateKey
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
		ForceGetGitDescribe:     false,
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
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("branch is empty, using the default branch %s", defaultBranch))
	}

	r, _ := regexp.Compile("{{.*}}")
	if commit != nil && commit.Value != "" && !r.MatchString(commit.Value) {
		opts.CheckoutCommit = commit.Value
	}

	var dir string
	if directory != nil {
		dir = directory.Value
	}

	workdir, err := workerruntime.WorkingDirectory(ctx)
	if err != nil {
		return sdk.Result{}, fmt.Errorf("Unable to find current working directory: %v", err)
	}

	workdirPath := workdir.Name()
	if x, ok := wk.BaseDir().(*afero.BasePathFs); ok {
		workdirPath, _ = x.RealPath(workdirPath)
	}

	return gitClone(ctx, wk, wk.Parameters(), gitURL, workdirPath, dir, auth, opts)
}

func gitClone(ctx context.Context, w workerruntime.Runtime, params []sdk.Parameter, url, basedir, dir string, auth *git.AuthOpts, clone *git.CloneOpts) (sdk.Result, error) {
	//Prepare all options - logs
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &git.OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	git.LogFunc = log.InfoWithoutCtx
	//Perform the git clone
	userLogCommand, err := git.Clone(url, basedir, dir, auth, clone, output)

	w.SendLog(ctx, workerruntime.LevelInfo, userLogCommand)

	//Send the logs
	if len(stdOut.Bytes()) > 0 {
		w.SendLog(ctx, workerruntime.LevelInfo, stdOut.String())
	}
	if len(stdErr.Bytes()) > 0 {
		w.SendLog(ctx, workerruntime.LevelWarn, stdErr.String())
	}

	if err != nil {
		return sdk.Result{}, fmt.Errorf("Unable to git clone: %s", err)
	}

	// extract info only if we git clone the same repo as current application linked to the pipeline
	gitURLSSH := sdk.ParameterValue(params, "git.url")
	gitURLHTTP := sdk.ParameterValue(params, "git.http_url")
	var vars []sdk.Variable
	if gitURLSSH == url || gitURLHTTP == url {
		vars, err = extractInfo(ctx, w, basedir, dir, params, clone.Tag, clone.Branch, clone.CheckoutCommit, clone)
	}

	if err != nil {
		w.SendLog(ctx, workerruntime.LevelWarn, err.Error())
	}

	stdTaglistErr := new(bytes.Buffer)
	stdTagListOut := new(bytes.Buffer)
	outputGitTag := &git.OutputOpts{
		Stderr: stdTaglistErr,
		Stdout: stdTagListOut,
	}

	errTag := git.TagList(url, basedir, dir, auth, outputGitTag)

	if len(stdTaglistErr.Bytes()) > 0 {
		w.SendLog(ctx, workerruntime.LevelInfo, stdTaglistErr.String())
	}

	if errTag != nil {
		return sdk.Result{}, sdk.WithStack(fmt.Errorf("Unable to list tag for getting current version: %v", errTag))
	}

	return sdk.Result{Status: sdk.StatusSuccess, NewVariables: vars}, nil
}

func extractInfo(ctx context.Context, w workerruntime.Runtime, basedir, dir string, params []sdk.Parameter, tag, branch, commit string, opts *git.CloneOpts) ([]sdk.Variable, error) {
	var res []sdk.Variable
	author := sdk.ParameterValue(params, "git.author")
	authorEmail := sdk.ParameterValue(params, "git.author.email")
	message := sdk.ParameterValue(params, "git.message")

	info, err := git.ExtractInfo(ctx, filepath.Join(dir), opts)
	if err != nil {
		return nil, err
	}

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

		var errS error
		cdsSemver, errS = computeSemver(info.GitDescribe, cdsVersion.Value)
		if errS != nil {
			w.SendLog(ctx, workerruntime.LevelError, errS.Error())
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
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.tag: %s", tag))
	} else {
		if branch == "" || branch == "{{.git.branch}}" {
			if info.Branch != "" {
				gitBranch := sdk.Variable{
					Name:  "git.branch",
					Type:  sdk.StringVariable,
					Value: info.Branch,
				}
				res = append(res, gitBranch)

				w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.branch: %s", info.Branch))
			} else {
				w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.branch: [empty]"))
			}
		} else if branch != "" && branch != "{{.git.branch}}" {
			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.branch: %s", branch))
		}

		if commit == "" || commit == "{{.git.hash}}" {
			if info.Hash != "" {
				res = append(res, sdk.Variable{
					Name:  "git.hash",
					Type:  sdk.StringVariable,
					Value: info.Hash,
				}, sdk.Variable{
					Name:  "git.hash.short",
					Type:  sdk.StringVariable,
					Value: sdk.StringFirstN(info.Hash, 7),
				})
				w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.hash: %s", info.Hash))
			} else {
				w.SendLog(ctx, workerruntime.LevelInfo, "git.hash: [empty]")
			}
		} else {
			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.hash: %s", commit))
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

			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.message: %s", info.Message))
		} else {
			w.SendLog(ctx, workerruntime.LevelInfo, "git.message: [empty]")
		}
	} else {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.message: %s", message))
	}

	if author == "" {
		if info.Author != "" {
			gitAuthor := sdk.Variable{
				Name:  "git.author",
				Type:  sdk.StringVariable,
				Value: info.Author,
			}

			res = append(res, gitAuthor)
			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.author: %s", info.Author))
		} else {
			w.SendLog(ctx, workerruntime.LevelInfo, "git.author: [empty]")
		}
	} else {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.author: %s", author))
	}

	if authorEmail == "" {
		if info.AuthorEmail != "" {
			gitAuthorEmail := sdk.Variable{
				Name:  "git.author.email",
				Type:  sdk.StringVariable,
				Value: info.AuthorEmail,
			}

			res = append(res, gitAuthorEmail)
			w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.author.email: %s", info.AuthorEmail))
		} else {
			w.SendLog(ctx, workerruntime.LevelInfo, "git.author.email: [empty]")
		}
	} else {
		w.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("git.author.email: %s", authorEmail))
	}
	return res, nil
}

func computeSemver(gitDescribe, cdsVersionValue string) (string, error) {
	var cdsSemver string
	smver, errT := semver.ParseTolerant(gitDescribe)
	if errT != nil {
		return "", fmt.Errorf("!! WARNING !! git describe %s is not semver compatible, we can't create cds.semver variable", gitDescribe)
	}
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
				cdsVersionValue,
			)
		} else if len(tuple) == 1 {
			cdsSemver = fmt.Sprintf("%d.%d.%d-%s+cds.%s",
				smver.Major,
				smver.Minor,
				smver.Patch,
				tuple[0],
				cdsVersionValue,
			)
		}
	}

	if cdsSemver == "" {
		// here, there is no prerelease version, it's a tag
		cdsSemver = fmt.Sprintf("%d.%d.%d+cds.%s",
			smver.Major,
			smver.Minor,
			smver.Patch,
			cdsVersionValue,
		)
	}

	// if git.describe contains a prefix 'v', we keep it
	if strings.HasPrefix(gitDescribe, "v") {
		cdsSemver = fmt.Sprintf("v%s", cdsSemver)
	}
	return cdsSemver, nil
}
