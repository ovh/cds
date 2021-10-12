package environment

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

// ImportOptions are options to import environment
type ImportOptions struct {
	Force          bool
	FromRepository string
}

// ParseAndImport parse an exportentities.Environment and insert or update the environment in database
func ParseAndImport(ctx context.Context, db gorpmapper.SqlExecutorWithTx, proj sdk.Project, eenv exportentities.Environment, opts ImportOptions, decryptFunc keys.DecryptFunc, u sdk.Identifiable) (*sdk.Environment, []sdk.Variable, []sdk.Message, error) {
	log.Debug(ctx, "ParseAndImport>> Import environment %s in project %s from repository %q (force=%v)", eenv.Name, proj.Key, opts.FromRepository, opts.Force)
	log.Debug(ctx, "ParseAndImport>> Env: %+v", eenv)

	msgList := []sdk.Message{}

	// Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(eenv.Name) {
		return nil, nil, nil, sdk.NewErrorFrom(sdk.ErrInvalidName, "environment name %s do not respect pattern %s", eenv.Name, sdk.NamePattern)
	}

	// Check if env exist
	oldEnv, err := LoadEnvironmentByName(db, proj.Key, eenv.Name)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrEnvironmentNotFound) {
		return nil, nil, nil, sdk.WrapError(err, "unable to load environment")
	}

	// If the environment exists and we don't want to force, raise an error
	var exist bool
	if oldEnv != nil && !opts.Force {
		return nil, nil, nil, sdk.WithStack(sdk.ErrEnvironmentExist)
	}
	if oldEnv != nil {
		exist = true
	}

	if oldEnv != nil {
		if opts.Force && opts.FromRepository == "" {
			if oldEnv.FromRepository != "" {
				if err := ascode.DeleteEventsEnvironmentOnlyFromRepoName(ctx, db, oldEnv.FromRepository, oldEnv.ID, oldEnv.Name); err != nil {
					return nil, nil, msgList, sdk.WrapError(err, "unable to delete as_code_event for %s on repo %s", oldEnv.Name, oldEnv.FromRepository)
				}
				msgList = append(msgList, sdk.NewMessage(sdk.MsgEnvironmentDetached, eenv.Name, oldEnv.FromRepository))
			}
			log.Debug(ctx, "ParseAndImport>> Force import environment %s in project %s without fromRepository", eenv.Name, proj.Key)
		} else if oldEnv.FromRepository != "" && opts.FromRepository != oldEnv.FromRepository {
			return nil, nil, nil, sdk.NewErrorFrom(sdk.ErrEnvironmentAsCodeOverride, "unable to update existing ascode environment from %s", oldEnv.FromRepository)
		}
	}

	env := new(sdk.Environment)
	env.Name = eenv.Name
	env.FromRepository = opts.FromRepository
	if exist {
		env.ID = oldEnv.ID
	}

	envSecrets := make([]sdk.Variable, 0)

	//Compute variables
	for p := range eenv.Values {
		value := eenv.Values[p].Value
		vtype := eenv.Values[p].Type

		switch vtype {
		case "":
			vtype = sdk.StringVariable
		case sdk.SecretVariable:
			secret, err := decryptFunc(ctx, db, proj.ID, value)
			if err != nil {
				return env, nil, nil, sdk.WrapError(err, "Unable to decrypt secret variable")
			}
			value = secret
		}

		vv := sdk.EnvironmentVariable{Name: p, Type: vtype, Value: value, EnvironmentID: env.ID}
		env.Variables = append(env.Variables, vv)

		if vtype == sdk.SecretVariable {
			envSecrets = append(envSecrets, sdk.Variable{
				Name:  fmt.Sprintf("cds.env.%s", vv.Name),
				Type:  vtype,
				Value: value,
			})
		}
	}

	//Compute keys
	for kname, kval := range eenv.Keys {
		if !strings.HasPrefix(kname, "env-") {
			return env, nil, nil, sdk.WrapError(sdk.ErrInvalidKeyName, "ParseAndImport>> Unable to parse key")
		}

		var oldKey *sdk.EnvironmentKey
		var keepOldValue bool
		//If env doesn't exist, skip the regen mecanism to generate key
		if oldEnv == nil {
			kval.Regen = nil
		} else {
			//If env exist, check the key exist
			oldKey = oldEnv.GetKey(kname)
			//If the key doesn't exist, skip the regen mecanism to generate key
			if oldKey == nil {
				kval.Regen = nil
			}
		}

		if kval.Regen != nil && !*kval.Regen {
			keepOldValue = true
		}

		kk, err := keys.Parse(ctx, db, proj.ID, kname, kval, decryptFunc)
		if err != nil {
			return env, nil, nil, sdk.WrapError(err, "Unable to parse key")
		}

		k := sdk.EnvironmentKey{
			EnvironmentID: env.ID,
			Name:          kname,
		}

		k.KeyID = kk.KeyID
		k.Public = kk.Public
		k.Private = kk.Private
		k.Type = kk.Type

		if keepOldValue && oldKey != nil {
			k.ID = oldKey.ID
			k.EnvironmentID = oldKey.EnvironmentID
			k.KeyID = oldKey.KeyID
			k.Public = oldKey.Public
			k.Private = oldKey.Private
			k.Type = oldKey.Type
		}

		env.Keys = append(env.Keys, k)

		envSecrets = append(envSecrets, sdk.Variable{
			Name:  fmt.Sprintf("cds.key.%s.priv", k.Name),
			Type:  string(k.Type),
			Value: k.Private,
		})
	}

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)

	go func(array *[]sdk.Message) {
		defer done.Done()
		for m := range msgChan {
			*array = append(*array, m)
		}
	}(&msgList)

	var globalError error

	if exist {
		globalError = ImportInto(ctx, db, env, oldEnv, msgChan, u)
	} else {
		globalError = Import(db, proj, env, msgChan, u)
	}

	close(msgChan)
	done.Wait()

	return env, envSecrets, msgList, globalError
}
