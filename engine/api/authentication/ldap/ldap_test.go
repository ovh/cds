package ldap

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

/*
  To setup the configuration of this unit test, you have to put it in the $HOME/.cds/api.tests.cfg.json file.
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
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
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

	driver, err := NewDriver(context.TODO(), false, ldapConfig)
	require.NoError(t, err)
	info, err := driver.GetUserInfo(context.TODO(), sdk.AuthConsumerSigninRequest{
		"bind":     cfg["ldapTestUsername"],
		"password": cfg["ldapTestPassword"],
	})

	require.NoError(t, err)
	require.Equal(t, cfg["ldapTestUsername"], info.Username)
	require.NotEmpty(t, info.Email, "Email")
	require.NotEmpty(t, info.Fullname, "Fullname")
	require.NotEmpty(t, info.ExternalID, "ExternalID")
}
