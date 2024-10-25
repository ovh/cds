package ldap

import (
	"context"
	"github.com/ovh/cds/engine/api/driver/ldap"
	"strconv"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
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
	log.Factory = log.NewTestingWrapper(t)
	cfg := test.LoadTestingConf(t, sdk.TypeAPI)
	ldapConfig := ldap.Config{
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
