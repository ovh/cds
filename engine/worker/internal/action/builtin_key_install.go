package action

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/afero"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func RunInstallKey(ctx context.Context, wk workerruntime.Runtime, a sdk.Action, params []sdk.Parameter, secrets []sdk.Variable) (sdk.Result, error) {
	var res sdk.Result
	keyName := sdk.ParameterFind(a.Parameters, "key")
	if keyName.Value == "" {
		return res, fmt.Errorf("Error: cannot have empty name for key parameter")
	}

	if secrets == nil {
		return res, fmt.Errorf("Cannot find any keys for your job")
	}

	var key *sdk.Variable
	for _, k := range secrets {
		if k.Name == ("cds.key." + keyName.Value + ".priv") {
			key = &k
			break
		}
	}

	if key == nil {
		return res, fmt.Errorf("Key %s not found", keyName.Value)
	}

	var filename string
	basePath, isBasePathFS := wk.Workspace().(*afero.BasePathFs)
	if isBasePathFS {
		realPath, _ := basePath.RealPath("/")
		filename = strings.TrimPrefix(filename, realPath)
		if runtime.GOOS == "darwin" {
			filename = strings.TrimPrefix(filename, "/private"+realPath)
		}
	}

	response, err := wk.InstallKey(*key, filename)
	if err != nil {
		log.Error("Unable to install key %s: %v", key.Name, err)
		if sdkerr, ok := err.(*sdk.Error); ok {
			return res, fmt.Errorf("%v", *sdkerr)
		} else {
			err := sdk.Error{
				Message: err.Error(),
				Status:  sdk.ErrUnknownError.Status,
			}
			return res, fmt.Errorf("Error: %v", err)
		}
	}

	switch response.Type {
	case sdk.KeyTypeSSH:
		if err := os.Setenv("PKEY", response.PKey); err != nil {
			return res, fmt.Errorf("Error: cannot export PKEY environment variable : %v", err)
		}
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Your SSH key '%s' is imported with success (%s)", keyName.Value, response.PKey))
	case sdk.KeyTypePGP:
		wk.SendLog(ctx, workerruntime.LevelInfo, fmt.Sprintf("Your PGP key '%s' is imported with success (%s)", keyName.Value, response.PKey))
	}

	return sdk.Result{
		Status: sdk.StatusSuccess,
	}, nil
}
