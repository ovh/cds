package environment

import (
	"strings"
	"sync"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// ImportOptions are options to import environment
type ImportOptions struct {
	Force          bool
	FromRepository string
}

// ParseAndImport parse an exportentities.Environment and insert or update the environment in database
func ParseAndImport(db gorp.SqlExecutor, proj *sdk.Project, eenv *exportentities.Environment, opts ImportOptions, decryptFunc keys.DecryptFunc, u *sdk.User) (*sdk.Environment, []sdk.Message, error) {
	log.Debug("ParseAndImport>> Import environment %s in project %s (force=%v)", eenv.Name, proj.Key, opts.Force)
	log.Debug("ParseAndImport>> Env: %+v", eenv)

	//Check valid application name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(eenv.Name) {
		return nil, nil, sdk.WrapError(sdk.ErrInvalidName, "ParseAndImport>> Environment name %s do not respect pattern %s", eenv.Name, sdk.NamePattern)
	}

	//Check if env exist
	oldEnv, errl := LoadEnvironmentByName(db, proj.Key, eenv.Name)
	if errl != nil && !sdk.ErrorIs(errl, sdk.ErrEnvironmentNotFound) {
		return nil, nil, sdk.WrapError(errl, "ParseAndImport>> Unable to load environment")
	}

	//If the environment exists and we don't want to force, raise an error
	var exist bool
	if oldEnv != nil && !opts.Force {
		return nil, nil, sdk.ErrEnvironmentExist
	}
	if oldEnv != nil {
		exist = true
	}

	if oldEnv != nil && oldEnv.FromRepository != "" && opts.FromRepository != oldEnv.FromRepository {
		return nil, nil, sdk.WrapError(sdk.ErrEnvironmentAsCodeOverride, "unable to update as code environment %s/%s.", oldEnv.FromRepository, opts.FromRepository)
	}

	env := new(sdk.Environment)
	env.Name = eenv.Name
	env.FromRepository = opts.FromRepository
	if exist {
		env.ID = oldEnv.ID
	}

	//Compute variables
	for p, v := range eenv.Values {
		switch v.Type {
		case "":
			v.Type = sdk.StringVariable
		case sdk.SecretVariable:
			secret, err := decryptFunc(db, proj.ID, v.Value)
			if err != nil {
				return env, nil, sdk.WrapError(err, "Unable to decrypt secret variable")
			}
			v.Value = secret
		}

		vv := sdk.Variable{Name: p, Type: v.Type, Value: v.Value}
		env.Variable = append(env.Variable, vv)
	}

	//Compute keys
	for kname, kval := range eenv.Keys {
		if !strings.HasPrefix(kname, "env-") {
			return env, nil, sdk.WrapError(sdk.ErrInvalidKeyName, "ParseAndImport>> Unable to parse key")
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

		kk, err := keys.Parse(db, proj.ID, kname, kval, decryptFunc)
		if err != nil {
			return env, nil, sdk.WrapError(err, "Unable to parse key")
		}

		k := sdk.EnvironmentKey{
			Key:           *kk,
			EnvironmentID: env.ID,
		}

		if keepOldValue && oldKey != nil {
			k.Key = oldKey.Key
		}

		env.Keys = append(env.Keys, k)
	}

	done := new(sync.WaitGroup)
	done.Add(1)
	msgChan := make(chan sdk.Message)
	msgList := []sdk.Message{}
	go func(array *[]sdk.Message) {
		defer done.Done()
		for m := range msgChan {
			*array = append(*array, m)
		}
	}(&msgList)

	var globalError error

	if exist {
		globalError = ImportInto(db, proj, env, oldEnv, msgChan, u)
	} else {
		globalError = Import(db, proj, env, msgChan, u)
	}

	close(msgChan)
	done.Wait()

	return env, msgList, globalError
}
