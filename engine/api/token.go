package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// generateTokenHandler allows a user to generate a token associated to a group permission
// and used by worker to take action from API.
// User generating the token needs to be admin of given group
func generateTokenHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	groupName := vars["permGroupName"]
	expiration := vars["expiration"]

	exp, err := sdk.ExpirationFromString(expiration)
	if err != nil {
		log.Warning("generateTokenHandler> '%s' -> %s\n", expiration, err)
		return err

	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("generateTokenHandler> cannot load group '%s': %s\n", groupName, err)
		return err

	}

	tk, err := worker.GenerateToken()
	if err != nil {
		log.Warning("generateTokenHandler: cannot generate key: %s\n", err)
		return err

	}

	if err := worker.InsertToken(db, g.ID, tk, exp); err != nil {
		log.Warning("generateTokenHandler> cannot insert new key: %s\n", err)
		return err

	}

	s := map[string]string{
		"key": tk,
	}
	return WriteJSON(w, r, s, http.StatusOK)
}
