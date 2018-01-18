package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/vcs"
	"github.com/ovh/cds/sdk/vcs/git"
)

func runGitTag(w *currentWorker) BuiltInAction {
	return func(ctx context.Context, a *sdk.Action, buildID int64, params *[]sdk.Parameter, sendLog LoggerFunc) sdk.Result {
		url := sdk.ParameterFind(&a.Parameters, "url")
		authPrivateKey := sdk.ParameterFind(&a.Parameters, "authPrivateKey")
		user := sdk.ParameterFind(&a.Parameters, "user")
		password := sdk.ParameterFind(&a.Parameters, "password")
		signKey := sdk.ParameterFind(&a.Parameters, "signKey")
		tagName := sdk.ParameterFind(&a.Parameters, "tagName")
		tagMessage := sdk.ParameterFind(&a.Parameters, "tagMessage")
		path := sdk.ParameterFind(&a.Parameters, "path")

		if tagName == nil || tagName.Value == "" {
			tagName = sdk.ParameterFind(params, "cds.semver")
			if tagName == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Tag name is not set. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
		}

		if url == nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "Git repository URL is not set. Nothing to perform.",
			}
			sendLog(res.Reason)
			return res
		}

		var username string
		if user == nil || user.Value == "" {
			u := sdk.ParameterFind(params, "cds.triggered_by.username")
			if u == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: "Git user is not set. Nothing to perform.",
				}
				sendLog(res.Reason)
				return res
			}
			username = u.Value
		} else {
			username = user.Value
		}

		if authPrivateKey != nil {
			//Setup the key
			if err := vcs.SetupSSHKey(nil, keysDirectory, authPrivateKey); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Unable to setup ssh key. %s", err),
				}
				sendLog(res.Reason)
				return res
			}
		}

		//Get the key
		key, errK := vcs.GetSSHKey(*params, keysDirectory, authPrivateKey)
		if errK != nil && errK != sdk.ErrKeyNotFound {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to setup ssh key. %s", errK),
			}
			sendLog(res.Reason)
			return res
		}

		//If url is not http(s), a key must be found
		if !strings.HasPrefix(url.Value, "http") {
			if errK == sdk.ErrKeyNotFound || key == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("SSH Key not found. Unable to perform git tag"),
				}
				sendLog(res.Reason)
				return res
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

		var msg = ""
		if tagMessage != nil {
			msg = tagMessage.Value
		}

		v, errT := semver.Make(tagName.Value)
		if errT != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Tag name is not semver compatible"),
			}
			sendLog(res.Reason)
			return res
		}
		v.Build = nil
		v.Pre = nil

		//Prepare all options - tag options
		var tagOpts = &git.TagOpts{
			Message:  msg,
			Name:     v.String(),
			Username: username,
		}

		if signKey != nil && signKey.Value != "" {
			privateKey := sdk.ParameterFind(params, fmt.Sprintf("%s.priv", signKey.Value))
			publicKey := sdk.ParameterFind(params, fmt.Sprintf("%s.pub", signKey.Value))
			keyID := sdk.ParameterFind(params, fmt.Sprintf("%s.id", signKey.Value))

			if privateKey == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Cannot find pgp private key."),
				}
				sendLog(res.Reason)
				return res
			}
			if keyID == nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Cannot find pgp key id."),
				}
				sendLog(res.Reason)
				return res
			}
			tagOpts.SignKey = privateKey.Value
			tagOpts.SignID = keyID.Value

			if err := ioutil.WriteFile("pgp.pub.key", []byte(publicKey.Value), 0600); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Cannot create pgp key file."),
				}
				sendLog(res.Reason)
				return res
			}
			if err := ioutil.WriteFile("pgp.key", []byte(tagOpts.SignKey), 0600); err != nil {
				res := sdk.Result{
					Status: sdk.StatusFail.String(),
					Reason: fmt.Sprintf("Cannot create pgp key file."),
				}
				sendLog(res.Reason)
				return res
			}
		}

		if key != nil {
			if auth == nil {
				auth = new(git.AuthOpts)
			}
			auth.PrivateKey = *key
		}

		//Prepare all options - logs
		stdErr := new(bytes.Buffer)
		stdOut := new(bytes.Buffer)

		output := &git.OutputOpts{
			Stderr: stdErr,
			Stdout: stdOut,
		}

		git.LogFunc = log.Info

		if path != nil {
			tagOpts.Path = path.Value
		}

		//Perform the git tag
		err := git.TagCreate(url.Value, auth, tagOpts, output)

		//Send the logs
		if len(stdOut.Bytes()) > 0 {
			sendLog(stdOut.String())
		}
		if len(stdErr.Bytes()) > 0 {
			sendLog(stdErr.String())
		}

		if err != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to git tag: %s", err),
			}
			sendLog(res.Reason)
			return res
		}

		semverVar := sdk.Variable{
			Name:  "cds.release.version",
			Type:  sdk.StringVariable,
			Value: tagOpts.Name,
		}
		_, errV := w.addVariableInPipelineBuild(semverVar, params)
		if errV != nil {
			res := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to save semver variable: %s", errV),
			}
			sendLog(res.Reason)
			return res
		}
		time.Sleep(5 * time.Second)
		return sdk.Result{Status: sdk.StatusSuccess.String()}
	}
}
