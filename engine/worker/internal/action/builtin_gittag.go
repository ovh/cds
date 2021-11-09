package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/blang/semver"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/vcs/git"
)

func RunGitTag(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, secrets []sdk.Variable) (sdk.Result, error) {
	tagPrerelease := sdk.ParameterFind(a.Parameters, "tagPrerelease")
	tagMetadata := sdk.ParameterFind(a.Parameters, "tagMetadata")
	tagLevel := sdk.ParameterFind(a.Parameters, "tagLevel")
	tagMessage := sdk.ParameterFind(a.Parameters, "tagMessage")
	path := sdk.ParameterFind(a.Parameters, "path")
	prefix := sdk.ParameterFind(a.Parameters, "prefix")

	tagLevelValid := true
	if tagLevel == nil || tagLevel.Value == "" {
		tagLevelValid = false
	} else if tagLevel.Value != "major" && tagLevel.Value != "minor" && tagLevel.Value != "patch" {
		tagLevelValid = false
	}

	if !tagLevelValid {
		return sdk.Result{}, errors.New("tag level is mandatory. It must be: 'major' or 'minor' or 'patch'")
	}

	gitURL, auth, err := vcsStrategy(ctx, wk, wk.Parameters(), secrets)
	if err != nil {
		return sdk.Result{}, err
	}

	var msg = ""
	if tagMessage != nil {
		msg = tagMessage.Value
	}

	cdsSemver := sdk.ParameterFind(wk.Parameters(), "cds.semver")
	if cdsSemver == nil || cdsSemver.Value == "" {
		return sdk.Result{}, errors.New("cds.semver is empty")
	}

	smver, errT := semver.ParseTolerant(cdsSemver.Value)
	if errT != nil {
		return sdk.Result{}, fmt.Errorf("cds.version '%s' is not semver compatible", cdsSemver.Value)
	}

	smver.Build = nil
	smver.Pre = nil

	switch tagLevel.Value {
	case "major":
		smver.Major++
		smver.Minor = 0
		smver.Patch = 0
	case "minor":
		smver.Minor++
		smver.Patch = 0
	default:
		smver.Patch++
	}

	r, _ := regexp.Compile(`^([0-9A-Za-z\-.]+)$`)
	// prerelease version notes: example: alpha, rc-1, ...
	if tagPrerelease != nil && tagPrerelease.Value != "" {
		if !r.MatchString(tagPrerelease.Value) {
			return sdk.Result{}, fmt.Errorf("tagPrerelease '%s' must comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-.]", tagPrerelease.Value)
		}
		smver.Pre = []semver.PRVersion{{VersionStr: tagPrerelease.Value}}
	}

	// metadata: this content is after '+'
	if tagMetadata != nil && tagMetadata.Value != "" {
		if !r.MatchString(tagMetadata.Value) {
			return sdk.Result{}, fmt.Errorf("tagMetadata '%s' must comprise only ASCII alphanumerics and hyphen [0-9A-Za-z-.]", tagMetadata.Value)
		}
		smver.Build = []string{tagMetadata.Value}
	}

	var userTag string
	userTrig := sdk.ParameterFind(wk.Parameters(), "cds.triggered_by.username")
	if userTrig != nil && userTrig.Value != "" {
		userTag = userTrig.Value
	} else {
		gitAuthor := sdk.ParameterFind(wk.Parameters(), "git.author")
		if gitAuthor != nil && gitAuthor.Value != "" {
			userTag = gitAuthor.Value
		}
	}

	if userTag == "" {
		return sdk.Result{}, fmt.Errorf("No user find to perform tag")
	}

	//Prepare all options - tag options
	var tagOpts = &git.TagOpts{
		Message:  msg,
		Name:     smver.String(),
		Username: userTag,
	}

	if prefix != nil && prefix.Value != "" {
		tagOpts.Name = fmt.Sprintf("%s%s", prefix.Value, tagOpts.Name)
	}

	if auth.SignKey.ID != "" {
		tagOpts.SignKey = auth.SignKey.Private
		tagOpts.SignID = auth.SignKey.ID

		if err := os.WriteFile("pgp.pub.key", []byte(auth.SignKey.Public), 0600); err != nil {
			return sdk.Result{}, fmt.Errorf("Cannot create pgp pub key file")
		}
		if err := os.WriteFile("pgp.key", []byte(tagOpts.SignKey), 0600); err != nil {
			return sdk.Result{}, fmt.Errorf("Cannot create pgp key file")
		}
	}

	// Run Git command

	//Prepare all options - logs
	stdErr := new(bytes.Buffer)
	stdOut := new(bytes.Buffer)

	output := &git.OutputOpts{
		Stderr: stdErr,
		Stdout: stdOut,
	}

	//git.LogFunc = log.InfoWithoutCtx

	if path != nil {
		tagOpts.Path = path.Value
	}

	//Perform the git tag
	err = git.TagCreate(gitURL, auth, tagOpts, output)

	//Send the logs
	if len(stdOut.Bytes()) > 0 {
		wk.SendLog(ctx, workerruntime.LevelInfo, stdOut.String())
	}
	if len(stdErr.Bytes()) > 0 {
		wk.SendLog(ctx, workerruntime.LevelWarn, stdErr.String())
	}

	if err != nil {
		return sdk.Result{}, fmt.Errorf("Unable to git tag: %v", err)
	}

	semverVar := sdk.Variable{
		Name:  "cds.release.version",
		Type:  sdk.StringVariable,
		Value: tagOpts.Name,
	}

	time.Sleep(5 * time.Second) // TODO: write here why we wait for 5 seconds
	return sdk.Result{
		Status:       sdk.StatusSuccess,
		NewVariables: []sdk.Variable{semverVar},
	}, nil
}
