package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// SharedInfraGroup is the group used to share infrastructure between projects
var (
	SharedInfraGroup *sdk.Group
	DefaultGroup     *sdk.Group
)

// CreateDefaultGroup creates a group 'public' where every user will be
func CreateDefaultGroup(db *gorp.DbMap, groupName string) error {
	if g, err := LoadByName(context.Background(), db, groupName); g != nil {
		return nil
	} else if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	var g = sdk.Group{
		Name: groupName,
	}
	if err := Insert(context.Background(), db, &g); err != nil {
		return err
	}

	return nil
}

// InitializeDefaultGroupName initializes sharedInfraGroup and Default Group
func InitializeDefaultGroupName(db gorp.SqlExecutor, defaultGrpName string) error {
	//Load the famous sharedInfraGroup
	var err error
	SharedInfraGroup, err = LoadByName(context.Background(), db, sdk.SharedInfraGroupName)
	if err != nil {
		return sdk.WrapError(err, "group.InitializeDefaultGroupName> Cannot load shared infra group")
	}

	if defaultGrpName != "" {
		DefaultGroup, err = LoadByName(context.Background(), db, defaultGrpName)
		if err != nil {
			return sdk.WrapError(err, "group.InitializeDefaultGroupName> Cannot load %s group", defaultGrpName)
		}
	}

	return nil
}

// IsDefaultGroupID returns true if groupID is the defaultGroupID
func IsDefaultGroupID(groupID int64) bool {
	if DefaultGroup == nil {
		return false
	}
	return groupID == DefaultGroup.ID
}
