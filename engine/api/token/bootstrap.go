package token

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Initialize inits token
func Initialize(db *gorp.DbMap, sharedInfraToken string) error {
	// check if shared infra has a token. If not, take token from config file.
	// if there is no token in config file -> it's an error
	nbToken, errt := CountToken(db, permission.SharedInfraGroupID)
	if errt != nil {
		return sdk.WrapError(errt, "Initialize> Cannot count token on default %s group", sdk.SharedInfraGroupName)
	}

	if nbToken > 0 {
		return nil
	}

	if len(sharedInfraToken) == 0 {
		return fmt.Errorf("Invalid Configuration. You have to set token for shared infra group in your configuration")
	}

	log.Info("Initialize> create token for %s group", sdk.SharedInfraGroupName)
	if err := InsertToken(db, permission.SharedInfraGroupID, sharedInfraToken, sdk.Persistent); err != nil {
		return sdk.WrapError(err, "Initialize> cannot insert new token for %s", sdk.SharedInfraGroupName)
	}

	return nil
}
