package accesstoken

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

// LoadSessionOptionFunc for auth session.
type LoadSessionOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthSession) error

// LoadSessionOptions provides all options on auth session loads functions.
var LoadSessionOptions = struct {
	WithGroups LoadSessionOptionFunc
}{
	WithGroups: loadGroups,
}

func loadGroups(ctx context.Context, db gorp.SqlExecutor, ss ...*sdk.AuthSession) error {
	// Get all group ids for auth sessions
	var groupIDs []int64
	for i := range ss {
		groupIDs = append(groupIDs, ss[i].GroupIDs...)
	}

	// Get all groups and create a map of group by ids
	gs, err := group.LoadAllByIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mGroups := make(map[int64]sdk.Group)
	for i := range gs {
		mGroups[gs[i].ID] = gs[i]
	}

	// Set groups for all given auth sessions
	for i := range ss {
		for j := range ss[i].GroupIDs {
			if g, ok := mGroups[ss[i].GroupIDs[j]]; ok {
				ss[i].Groups = append(ss[i].Groups, g)
			}
		}
	}

	return nil
}
