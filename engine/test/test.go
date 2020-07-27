package test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"testing"

	"github.com/ovh/cds/sdk/log"
)

// LoadTestingConf loads test configuration tests.cfg.json
func LoadTestingConf(t log.Logger, serviceType string) map[string]string {
	var f string
	u, _ := user.Current()
	if u != nil {
		f = path.Join(u.HomeDir, ".cds", serviceType+".tests.cfg.json")
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

//GetTestName returns the name the test
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
