package config

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/stretchr/testify/require"
)

// LoadTestingConf loads test configuration tests.cfg.json
func LoadTestingConf(t require.TestingT, serviceType string) map[string]string {
	var f string
	u, _ := user.Current()
	if u != nil {
		f = path.Join(u.HomeDir, ".cds", serviceType+".tests.cfg.json")
	}

	_, err := os.Stat(f)
	require.NoError(t, err, "error no test configuration file found at %s", f)

	btes, err := ioutil.ReadFile(f)
	require.NoError(t, err, "error reading test configuration file from %s", f)
	if len(btes) != 0 {
		cfg := map[string]string{}
		require.NoError(t, json.Unmarshal(btes, &cfg), "error to unmarshal test configuration file from %s", f)
		return cfg
	}

	return nil
}
