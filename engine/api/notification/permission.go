package notification

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
)

// projectPermissionUsers Get users that access to given project, without default group
func projectPermissionUserIDs(ctx context.Context, db gorp.SqlExecutor, store cache.Store, projectID int64, access int) ([]string, error) {
	proj, err := project.LoadByID(db, projectID, project.LoadOptions.WithGroups)
	if err != nil {
		return nil, err
	}

	var groupIDs []int64
	for _, g := range proj.ProjectGroups {
		if group.DefaultGroup != nil && group.DefaultGroup.ID == g.Group.ID {
			continue // we don't want to sent notif on all user to the default group
		}
		groupIDs = append(groupIDs, g.Group.ID)
	}

	grps, err := group.LoadAllByIDs(ctx, db, groupIDs, group.LoadOptions.WithMembers)
	if err != nil {
		return nil, err
	}

	var userIDsMap = make(map[string]struct{})
	var userIDs []string
	for _, g := range grps {
		for _, m := range g.Members {
			if _, has := userIDsMap[m.ID]; !has {
				userIDsMap[m.ID] = struct{}{}
				userIDs = append(userIDs, m.ID)
			}
		}
	}

	return userIDs, nil
}
