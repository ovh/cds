package repositoriesmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/repositoriesmanager/repogithub"
	"github.com/ovh/cds/engine/api/repositoriesmanager/repostash"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	initialized bool
	options     InitializeOpts
)

//InitializeOpts is the struct to init the package
type InitializeOpts struct {
	KeysDirectory          string
	UIBaseURL              string
	APIBaseURL             string
	DisableStashSetStatus  bool
	DisableGithubSetStatus bool
	DisableGithubStatusURL bool
	GithubSecret           string
	StashPrivateKey        string
	StashConsumerKey       string
}

//Initialize initialize private keys
//For instance for a repositories manager named "github.com/ovh", the private key
func Initialize(o InitializeOpts) error {
	options = o
	repogithub.Init(o.APIBaseURL, o.UIBaseURL)
	repostash.Init(o.APIBaseURL, o.UIBaseURL)

	_db := database.DB()
	if _db == nil {
		return fmt.Errorf("Unable to init repositories manager")
	}
	if db := database.DBMap(_db); db != nil {
		repositoriesManager, err := LoadAll(db)
		if err != nil {
			return err
		}
		for _, rm := range repositoriesManager {
			var found bool
			rmSecrets := map[string]string{}

			switch rm.Type {
			case sdk.Stash:
				if o.StashPrivateKey != "" {
					log.Info("RepositoriesManager> Found a key for %s", rm.Name)
					rmSecrets["privatekey"] = o.StashPrivateKey
					found = true
				}
			case sdk.Github:
				if o.GithubSecret != "" {
					log.Info("RepositoriesManager> Found a client-secret for %s", rm.Name)
					// rmSecrets does not need to contains ["client-secret"] = o.GithubSecret
					// GithubSecret is already the real secret, not a path to a file
					found = true
				}
			}

			if found {
				if err := initRepositoriesManager(db, &rm, o.KeysDirectory, rmSecrets); err != nil {
					log.Warning("RepositoriesManager> Unable init %s", rm.Name)
				}
			} else {
				log.Warning("RepositoriesManager> Unable to find key for %s", rm.Name)
			}
		}
		initialized = true
		return nil
	}
	return errors.New("Cannot init repositories manager")
}

//New instanciate a new RepositoriesManager, act as a Factory with all supported repositories manager
func New(t sdk.RepositoriesManagerType, id int64, name, URL string, args map[string]string, consumerData string) (*sdk.RepositoriesManager, error) {
	switch t {
	case sdk.Stash:
		//we have to compute the StashConsumer
		var stash *repostash.StashConsumer
		//Check if it isn't coming from the DB
		if id == 0 || consumerData == "" {
			//Check args
			if len(args) != 1 || args["key"] == "" {
				return nil, fmt.Errorf("key args is mandatory to connect to stash")
			}
			stash = repostash.New(URL, options.StashConsumerKey, args["key"])
		} else {
			//It's coming from the database, we just have to unmarshal data from the DB to get consumerData
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(consumerData), &data); err != nil {
				log.Warning("New> Error %s", err)
				return nil, err
			}
			stash = repostash.New(URL, data["consumer_key"].(string), data["private_rsa_key"].(string))
		}
		stash.DisableSetStatus = options.DisableStashSetStatus

		if stash.DisableSetStatus {
			log.Debug("RepositoriesManager> ⚠ Stash Statuses are disabled")
		}

		rm := sdk.RepositoriesManager{
			ID:               id,
			Consumer:         stash,
			Name:             name,
			URL:              URL,
			Type:             sdk.Stash,
			HooksSupported:   stash.HooksSupported(),
			PollingSupported: stash.PollingSupported(),
		}
		return &rm, nil
	case sdk.Github:
		var github *repogithub.GithubConsumer
		var withHook, withPolling *bool

		//Check if it isn't coming from the DB
		if id == 0 || consumerData == "" {
			//Check args
			if len(args) < 1 || args["client-id"] == "" || options.GithubSecret == "" {
				return nil, fmt.Errorf("client-id args and client-secret (in cds configuration) are mandatory to connect to github : %v", args)
			}

			github = repogithub.New(args["client-id"], options.GithubSecret, options.APIBaseURL+"/repositories_manager/oauth2/callback")
			if args["with-hooks"] != "" {
				b, err := strconv.ParseBool(args["with-hooks"])
				if err == nil {
					withHook = &b
				}
			}

			if args["with-polling"] != "" {
				b, err := strconv.ParseBool(args["with-polling"])
				if err == nil {
					withPolling = &b
				}
			}
		} else {
			//It's coming from the database, we just have to unmarshal data from the DB to get consumerData
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(consumerData), &data); err != nil {
				log.Warning("New> Error %s", err)
				return nil, err
			}

			github = repogithub.New(data["client-id"].(string), options.GithubSecret, options.APIBaseURL+"/repositories_manager/oauth2/callback")
			if data["with-hooks"] != nil {
				b, ok := data["with-hooks"].(bool)
				if !ok {
					b = github.HooksSupported()
				}
				withHook = &b
			}

			if data["with-polling"] != nil {
				b, ok := data["with-polling"].(bool)
				if !ok {
					b = github.PollingSupported()
				}
				withPolling = &b
			}
		}

		github.DisableSetStatus = options.DisableGithubSetStatus
		github.DisableStatusURL = options.DisableGithubStatusURL

		if withHook == nil {
			log.Debug("with hooks : default")
			b := github.HooksSupported()
			withHook = &b
		}
		github.WithHooks = *withHook
		if withPolling == nil {
			log.Debug("with polling : default")
			b := github.PollingSupported()
			withPolling = &b
		}
		github.WithPolling = *withPolling

		rm := sdk.RepositoriesManager{
			ID:               id,
			Consumer:         github,
			Name:             name,
			URL:              repogithub.URL,
			Type:             sdk.Github,
			HooksSupported:   *withHook && github.HooksSupported(),
			PollingSupported: *withPolling && github.PollingSupported(),
		}

		return &rm, nil
	}
	return nil, fmt.Errorf("Unknown type %s. Cannot instanciate repositories manager t=%s id=%d name=%s url=%s args=%s consumerData=%s", t, t, id, name, URL, args, consumerData)
}

//Init initializes all repositories
func initRepositoriesManager(db gorp.SqlExecutor, rm *sdk.RepositoriesManager, directory string, secrets map[string]string) error {
	if rm.Type == sdk.Stash {
		privateKey := secrets["privatekey"]
		if privateKey == "" {
			return fmt.Errorf("Cannot init %s. Missing private key", privateKey)
		}
		path := filepath.Join(directory, fmt.Sprintf("%s.%s", rm.Name, "privateKey"))
		log.Info("RepositoriesManager> Writing stash private key %s", path)
		if err := ioutil.WriteFile(path, []byte(privateKey), 0600); err != nil {
			log.Warning("RepositoriesManager> Unable to write stash private key %s : %s", path, err)
			return err
		}
		stash := rm.Consumer.(*repostash.StashConsumer)
		stash.PrivateRSAKey = path
		if err := Update(db, rm); err != nil {
			return err
		}
		return nil
	}

	if rm.Type == sdk.Github {
		// nothing to do here for github
		return nil
	}
	return fmt.Errorf("Unsupported repositories manager : %s: %s", rm.Name, rm.Type)
}
