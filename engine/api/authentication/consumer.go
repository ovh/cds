package authentication

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func NewConsumerWorker(db gorp.SqlExecutor, name string, hatcherySrv *sdk.Service, hatcheryConsumer *sdk.AuthConsumer, groupIDs []int64) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               name,
		AuthentifiedUserID: hatcheryConsumer.AuthentifiedUserID,
		ParentID:           &hatcheryConsumer.ID,
		Type:               sdk.ConsumerBuiltin,
		Data:               map[string]string{},
		GroupIDs:           groupIDs,
		Scopes: []sdk.AuthConsumerScope{
			sdk.AuthConsumerScopeWorker,
			sdk.AuthConsumerScopeWorkerModel,
			sdk.AuthConsumerScopeRun,
			sdk.AuthConsumerScopeRunExecution,
		},
		IssuedAt: time.Now(),
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// NewConsumerExternal returns a new local consumer for given data.
func NewConsumerExternal(db gorp.SqlExecutor, userID string, consumerType sdk.AuthConsumerType, userInfo sdk.AuthDriverUserInfo) (*sdk.AuthConsumer, error) {
	c := sdk.AuthConsumer{
		Name:               string(consumerType),
		AuthentifiedUserID: userID,
		Type:               consumerType,
		Data: map[string]string{
			"external_id": userInfo.ExternalID,
			"fullname":    userInfo.Fullname,
			"username":    userInfo.Username,
			"email":       userInfo.Email,
		},
		IssuedAt: time.Now(),
	}

	if err := InsertConsumer(db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// ConsumerRegen updates a consumer issue date to invalidate old signin token.
func ConsumerRegen(db gorp.SqlExecutor, consumer *sdk.AuthConsumer) error {
	if consumer.Type != sdk.ConsumerBuiltin {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't regen a no builtin consumer")
	}
	if consumer.Disabled {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't regen a disabled consumer")
	}

	// Remove invalid groups and warnings
	consumer.InvalidGroupIDs = nil
	consumer.Warnings = nil

	// Update the IAT attribute in database
	consumer.IssuedAt = time.Now()
	if err := UpdateConsumer(db, consumer); err != nil {
		return err
	}

	return nil
}

// ConsumerRemoveGroup removes given group from all consumers that using it, set warning and disabled state if needed.
func ConsumerRemoveGroup(ctx context.Context, db gorp.SqlExecutor, g *sdk.Group) error {
	// Load all consumers that refer to the group
	cs, err := LoadConsumersByGroupID(ctx, db, g.ID)
	if err != nil {
		return err
	}
	for i := range cs {
		// Remove the group id from the consumer and add a warning to the consumer
		if !cs[i].GroupIDs.Contains(g.ID) && !cs[i].InvalidGroupIDs.Contains(g.ID) {
			continue
		}

		cs[i].GroupIDs.Remove(g.ID)
		cs[i].InvalidGroupIDs.Remove(g.ID)

		// Clean warnings, removes warning for invalid group on given one
		filteredWarnings := make(sdk.AuthConsumerWarnings, 0, len(cs[i].Warnings))
		for _, w := range cs[i].Warnings {
			if !(w.Type == sdk.WarningGroupInvalid && w.GroupID == g.ID) {
				filteredWarnings = append(filteredWarnings, w)
			}
		}
		cs[i].Warnings = filteredWarnings

		// Add a new warning for group removed
		cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningGroupRemoved(g.ID, g.Name))

		// If there is no group left in the consumer we want to disable it if not already disabled
		if len(cs[i].GroupIDs) == 0 && !cs[i].Disabled {
			cs[i].Disabled = true
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningLastGroupRemoved())
		}

		if err := UpdateConsumer(db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerInvalidateGroupForUser set group as invalid in all user's consumers and set warning.
func ConsumerInvalidateGroupForUser(ctx context.Context, db gorp.SqlExecutor, g *sdk.Group, userID string) error {
	// Load all consumers for the user
	cs, err := LoadConsumersByUserID(ctx, db, userID)
	if err != nil {
		return err
	}
	for i := range cs {
		if !cs[i].GroupIDs.Contains(g.ID) {
			continue
		}

		// Remove the group id from slice and add it to the invalid ones
		cs[i].GroupIDs.Remove(g.ID)
		cs[i].InvalidGroupIDs = append(cs[i].InvalidGroupIDs, g.ID)
		cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningGroupInvalid(g.ID, g.Name))

		// If there is no group left in the consumer we want to disable it
		if len(cs[i].GroupIDs) == 0 {
			cs[i].Disabled = true
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningLastGroupRemoved())
		}

		if err := UpdateConsumer(db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerRestoreInvalidatedGroupForUser checks if there are consumers for given user where the group was invalidated, then
// restore it and remove warning.
func ConsumerRestoreInvalidatedGroupForUser(ctx context.Context, db gorp.SqlExecutor, groupID int64, userID string) error {
	// Load all consumers for the user
	cs, err := LoadConsumersByUserID(ctx, db, userID)
	if err != nil {
		return err
	}
	for i := range cs {
		if !cs[i].InvalidGroupIDs.Contains(groupID) {
			continue
		}

		// Remove the group id from slice and add it to the valid ones
		cs[i].InvalidGroupIDs.Remove(groupID)
		cs[i].GroupIDs = append(cs[i].GroupIDs, groupID)

		// If the consumer was disabled because there was no group left inside, it can be re-enable
		cs[i].Disabled = false

		// Clean warnings, removes warning for current group and last group removed wanring if exists
		filteredWarnings := make(sdk.AuthConsumerWarnings, 0, len(cs[i].Warnings))
		for _, w := range cs[i].Warnings {
			if (w.Type == sdk.WarningGroupInvalid && w.GroupID != groupID) ||
				w.Type == sdk.WarningGroupRemoved {
				filteredWarnings = append(filteredWarnings, w)
			}
		}
		cs[i].Warnings = filteredWarnings

		if err := UpdateConsumer(db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}
