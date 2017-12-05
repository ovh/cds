package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"testing"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//DBDriver is exported for testing purpose
var (
	DBDriver      string
	dbUser        string
	dbPassword    string
	dbName        string
	dbHost        string
	dbPort        int64
	dbSSLMode     string
	RedisHost     string
	RedisPassword string
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

type Bootstrapf func(sdk.DefaultValues, func() *gorp.DbMap) error

var DBConnectionFactory *database.DBConnectionFactory

// SetupPG setup PG DB for test
func SetupPG(t *testing.T, bootstrapFunc ...Bootstrapf) (*gorp.DbMap, cache.Store) {
	log.SetLogger(t)
	cfg := LoadTestingConf(t)
	DBDriver = cfg["dbDriver"]
	dbUser = cfg["dbUser"]
	dbPassword = cfg["dbPassword"]
	dbName = cfg["dbName"]
	dbHost = cfg["dbHost"]
	var err error
	dbPort, err = strconv.ParseInt(cfg["dbPort"], 10, 64)
	if err != nil {
		t.Errorf("Error when unmarshal config %s", err)
	}
	dbSSLMode = cfg["sslMode"]
	RedisHost = cfg["redisHost"]
	RedisPassword = cfg["redisPassword"]

	secret.Init("3dojuwevn94y7orh5e3t4ejtmbtstest")

	if DBDriver == "" {
		t.Fatalf("This should be run with a database")
		return nil, nil
	}
	if DBConnectionFactory == nil {
		var err error
		DBConnectionFactory, err = database.Init(dbUser, dbPassword, dbName, dbHost, int(dbPort), dbSSLMode, 10, 2000, 100)
		if err != nil {
			t.Fatalf("Cannot open database: %s", err)
			return nil, nil
		}
	}

	for _, f := range bootstrapFunc {
		if err := f(sdk.DefaultValues{SharedInfraToken: sdk.RandomString(32)}, DBConnectionFactory.GetDBMap); err != nil {
			log.Error("Error: %v", err)
			return nil, nil
		}
	}

	store, err := cache.NewRedisStore(RedisHost, RedisPassword, 60)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}
	event.Cache = store
	pipeline.Store = store

	return DBConnectionFactory.GetDBMap(), store
}

// LoadTestingConf loads test configuraiton tests.cfg.json
func LoadTestingConf(t *testing.T) map[string]string {
	var f string
	u, _ := user.Current()
	if u != nil {
		f = path.Join(u.HomeDir, ".cds", "tests.cfg.json")
	}

	if _, err := os.Stat(f); err == nil {
		t.Logf("Tests configuration read from %s", f)
		btes, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("Error reading %s: %v", f, err)
		}
		if len(btes) != 0 {
			cfg := map[string]string{}
			if err := json.Unmarshal(btes, &cfg); err != nil {
				t.Fatalf("Error reading %s: %v", f, err)
			}
			return cfg
		}
	} else {
		t.Fatalf("Error reading %s: %v", f, err)
	}
	return nil
}

//GetTestName returns the name the the test
func GetTestName(t *testing.T) string {
	return t.Name()
}

//FakeHTTPClient implements sdk.HTTPClient and returns always the same response
type FakeHTTPClient struct {
	T        *testing.T
	Response *http.Response
	Error    error
}

//Do implements sdk.HTTPClient and returns always the same response
func (f *FakeHTTPClient) Do(r *http.Request) (*http.Response, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err == nil {
		r.Body.Close()
	}

	f.T.Logf("FakeHTTPClient> Do> %s %s: Payload %s", r.Method, r.URL.String(), string(b))
	return f.Response, f.Error
}
