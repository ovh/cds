package ldap

import (
	"strconv"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk/log"
)

/*
  To setup the configuration of this unit test, you have to put it in the $HOME/.cds/tests.cfg.json file.
  Add the following attributes after the database configuration and set as you want
  	"ldapRootDN" : "",
    "ldapUserSearchBase" : "",
    "ldapUserSearch" : "",
    "ldapFullname" : "",
    "ldapHost" : "",
    "ldapPort" : "",
    "ldapSSL" : "",
    "ldapTestUsername": "",
    "ldapTestPassword": "",
    "ldapManagerDN": "",
	"ldapManagerPassword": ""

  If ldapHost is not set, the test if skipped
*/

func TestGetUserInfo(t *testing.T) {
	log.SetLogger(t)
	cfg := test.LoadTestingConf(t)
	ldapConfig := Config{
		RootDN:          cfg["ldapRootDN"],
		UserSearchBase:  cfg["ldapUserSearchBase"],
		UserSearch:      cfg["ldapUserSearch"],
		UserFullname:    cfg["ldapFullname"],
		Host:            cfg["ldapHost"],
		ManagerDN:       cfg["ldapManagerDN"],
		ManagerPassword: cfg["ldapManagerPassword"],
	}

	if ldapConfig.Host == "" {
		t.SkipNow()
	}

	ldapConfig.Port, _ = strconv.Atoi(cfg["ldapPort"])
	ldapConfig.SSL, _ = strconv.ParseBool(cfg["ldapSSL"])

	driver, err := NewDriver(false, ldapConfig)
	require.NoError(t, err)
	info, err := driver.GetUserInfo(sdk.AuthConsumerSigninRequest{
		"bind":     cfg["ldapTestUsername"],
		"password": cfg["ldapTestPassword"],
	})

	require.NoError(t, err)
	require.Equal(t, cfg["ldapTestUsername"], info.Username)
	require.NotEmpty(t, info.Email, "Email")
	require.NotEmpty(t, info.Fullname, "Fullname")
	require.NotEmpty(t, info.ExternalID, "ExternalID")
}
