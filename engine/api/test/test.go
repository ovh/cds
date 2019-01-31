package test

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"strconv"
	"testing"

	"github.com/ovh/cds/engine/api/accesstoken"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//DBDriver is exported for testing purpose
var (
	DBDriver      string
	dbUser        string
	dbRole        string
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

// This is a test key, do not use it in real life
var TestKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEogIBAAKCAQEAj3YCi33CaIiWfhsYz3lxOGjDSxxtA+LM4dDjIFe3Xq+gntcg
1WKFoAsnHFgC3sOoZKSeIjuBIsGXvfOzOs10EdlU388bAOP51NmsGLtVwBSpYQkQ
FGb1QricuZy6BZB0JiBM9raz5ikszG3m52opS3pibw19ZyvUSSjHAiXEaJpML0m/
YiKowrf2bO2cFbSATCDEhK5pDhzllRhLOkST/VH3QSrKL0xydKNjGmmJDlpM2xKT
7Vbb2DkMPl4kVnYf/XveojS0GSbsQaIS17WEMayP4ch9g27E5GMp0+IZ7w9Dq/ai
7T+hMqlkFfajB97zTqHFRD4hMITckjpPlPx8WwIDAQABAoIBABpC8xJP8i+qmUn6
cd9BDu3Rl7Z/PKGSegj4cStxgzrNEa0iGnuVbnqur/krT1MyI/hQfjYsCGaxY7K9
Etk31QCTdUsHIZ1XHlvNgQiB+p+P6LW/r/bcJheRrfb4bsEoAWsdTJl5NpNyhCXk
FHnWYDrV64ECyisBxfmiglOtUDgJht1IKgIp9vULWJPQ7/PYRc7R7kpfiSGTPgmP
LV/20edWfBsbxPR/2rL5azpL3YIkJgNDRrnieDHzuOJ86FzICWq8gLhta9j5FARG
0PSs0Myy9ucfAu+lVi4S5/GsyfEiljXznGyQxFwR9EZp+BZEBvdFtkIod48d0DQy
t7xmrEECgYEA8KU+9O1pC/B8/61ZLLMEgq7EQ1qUDZ5cNSIJoTf9vvCQD+hCbtwY
Wgq+MIYR0dNn8MxAmwsZeAFfu9USJNiDKzc7yYSQ4OJXHXk33895UllsZdpaj5cc
hGkxnr8JMdWLIsmeCF8F9mIQywV+QLmjPQVW8VBYFY4+0dbrfmtIYSECgYEAmJ1U
6klHtEWv+Msc8Yjg/d5oPQuBy9ilRv97g5ilaHQ4aMDvsiV1HCxER0NA85jjCP+/
ulYwoLWgV+WObbEGeg+B929oHRSFp/XTvEWhoOxAAICMrVwQ6qX5yPOAKtPKmkop
m6PbzM+QrIRw0cYXEZEVG3Cme8x+sHKQ54CAIfsCgYBGzig3Ar+8zpbI1+V8HHRA
S1HeC4GyfBzfWVOCByp3CusocwtQ+RuFKtIJDvmhRlW36TE9LUfiIm1bo/bBtp7p
kUfbJFFIifBd8LO6+53T2BHn6hZpV2oBn74E2mrHKfDVXINOLT9g3jvYsJYUT0qz
gqWxPRWdygu7zEPgH4rdYQKBgBzvDy9P71k9MQyhLX6ZbdaTuP2B1fzYuRUJ0Nf1
M77m8d7iXU9QDLDnr5Y3KPRGEx0cp7PjLVr6tEiVy/f97PVtRT2tEHca8fATCi6S
oP8Ka2Ps+z7OyqJCD2ZKzAzSlIHF97d7TGu7Gnmqrl0HCk6ZTAAkzluAPLClN9W8
Jg7LAoGAFxXOBXuGB+Lsbgioka0vM1mGYWEKjobPcQRkMq37b6GdkhMl2A5fH4C+
uhOrSSJ8cK0UO9ET6DV6V5MuQoEAMVYt8v39fxOnrH7sX2OwTqXOqK7b27vfcY+g
G6f1bOI7lNhA4uAqZICcXO8cxwEa8xoeuPFT2I0R8tzAD5GhIto=
-----END RSA PRIVATE KEY-----`)

// SetupPG setup PG DB for test
func SetupPG(t log.Logger, bootstrapFunc ...Bootstrapf) (*gorp.DbMap, cache.Store, context.CancelFunc) {
	log.SetLogger(t)
	cfg := LoadTestingConf(t)
	DBDriver = cfg["dbDriver"]
	dbUser = cfg["dbUser"]
	dbRole = cfg["dbRole"]
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
	accesstoken.Init("cds_test", TestKey) // nolint

	if DBDriver == "" {
		t.Fatalf("This should be run with a database")
		return nil, nil, func() {}
	}
	if DBConnectionFactory == nil {
		var err error
		DBConnectionFactory, err = database.Init(dbUser, dbRole, dbPassword, dbName, dbHost, int(dbPort), dbSSLMode, 10, 2000, 100)
		if err != nil {
			t.Fatalf("Cannot open database: %s", err)
			return nil, nil, func() {}
		}
	}

	for _, f := range bootstrapFunc {
		if err := f(sdk.DefaultValues{SharedInfraToken: sdk.RandomString(32)}, DBConnectionFactory.GetDBMap); err != nil {
			log.Error("Error: %v", err)
			return nil, nil, func() {}
		}
	}

	store, err := cache.NewRedisStore(RedisHost, RedisPassword, 60)
	if err != nil {
		t.Fatalf("Unable to connect to redis: %v", err)
	}

	cancel := func() {
		store.Client.Close()
		store.Client = nil
	}

	return DBConnectionFactory.GetDBMap(), store, cancel
}

// LoadTestingConf loads test configuraiton tests.cfg.json
func LoadTestingConf(t log.Logger) map[string]string {
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
