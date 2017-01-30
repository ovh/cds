package worker

import (
	"fmt"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/group"
)

//Initialize init the package
func Initialize() error {
	go Heartbeat()
	go ModelCapabilititiesCacheLoader(5)

	db := database.DB()
	if db == nil {
		return fmt.Errorf("Database is unavailabe")
	}

	g, err := group.LoadGroup(database.DBMap(db), group.SharedInfraGroup)
	if err != nil {
		return err
	}

	sharedInfraGroupID = g.ID
	return nil
}

var (
	sharedInfraGroupID int64
)
