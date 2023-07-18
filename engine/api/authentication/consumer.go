package authentication

import (
	"context"
	"time"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func NewConsumerHatchery(ctx context.Context, db gorpmapper.SqlExecutorWithTx, h sdk.Hatchery) (*sdk.AuthHatcheryConsumer, error) {
	c := sdk.AuthHatcheryConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            h.Name,
			Type:            sdk.ConsumerHatchery,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 365*24*time.Hour),
			Description:     "Consumer for hatchery " + h.Name,
		},
		AuthConsumerHatchery: sdk.AuthConsumerHatcheryData{
			HatcheryID: h.ID,
		},
	}
	if err := InsertHatcheryConsumer(ctx, db, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func NewConsumerWorker(ctx context.Context, db gorpmapper.SqlExecutorWithTx, name string, hatcheryConsumer *sdk.AuthUserConsumer) (*sdk.AuthUserConsumer, error) {
	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            name,
			ParentID:        &hatcheryConsumer.ID,
			Type:            sdk.ConsumerBuiltin,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 24*time.Hour),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: hatcheryConsumer.AuthConsumerUser.AuthentifiedUserID,
			Data:               map[string]string{},
			ScopeDetails: sdk.NewAuthConsumerScopeDetails(
				sdk.AuthConsumerScopeRunExecution,
			),
		},
	}

	if err := InsertUserConsumer(ctx, db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

func NewConsumerWorkerV2(ctx context.Context, db gorpmapper.SqlExecutorWithTx, name string, hatcheryConsumer *sdk.AuthHatcheryConsumer) (*sdk.AuthHatcheryConsumer, error) {
	c := sdk.AuthHatcheryConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            name,
			ParentID:        &hatcheryConsumer.ID,
			Type:            sdk.ConsumerHatchery,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 24*time.Hour),
		},
		AuthConsumerHatchery: sdk.AuthConsumerHatcheryData{
			HatcheryID: hatcheryConsumer.AuthConsumerHatchery.HatcheryID,
		},
	}

	if err := InsertHatcheryConsumer(ctx, db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// NewConsumerExternal returns a new local consumer for given data.
func NewConsumerExternal(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID string, consumerType sdk.AuthConsumerType, userInfo sdk.AuthDriverUserInfo) (*sdk.AuthUserConsumer, error) {
	c := sdk.AuthUserConsumer{
		AuthConsumer: sdk.AuthConsumer{
			Name:            string(consumerType),
			Type:            consumerType,
			ValidityPeriods: sdk.NewAuthConsumerValidityPeriod(time.Now(), 0),
		},
		AuthConsumerUser: sdk.AuthUserConsumerData{
			AuthentifiedUserID: userID,
			Data: map[string]string{
				"external_id": userInfo.ExternalID,
				"fullname":    userInfo.Fullname,
				"username":    userInfo.Username,
				"email":       userInfo.Email,
			},
		},
	}

	if err := InsertUserConsumer(ctx, db, &c); err != nil {
		return nil, err
	}

	return &c, nil
}

// ConsumerRegen updates a consumer issue date to invalidate old signin token.
func ConsumerRegen(ctx context.Context, db gorpmapper.SqlExecutorWithTx, consumer *sdk.AuthUserConsumer, overlapDuration, newDuration time.Duration) error {
	if consumer.Type != sdk.ConsumerBuiltin {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't regen a no builtin consumer")
	}
	if consumer.Disabled {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "can't regen a disabled consumer")
	}

	consumer.AuthConsumerUser.InvalidGroupIDs = nil
	consumer.Warnings = nil

	// Regen the token
	latestPeriod := consumer.ValidityPeriods.Latest()
	latestPeriod.Duration = time.Now().Add(overlapDuration).Sub(latestPeriod.IssuedAt)
	consumer.ValidityPeriods = append(consumer.ValidityPeriods,
		sdk.AuthConsumerValidityPeriod{
			IssuedAt: time.Now(),
			Duration: newDuration,
		},
	)
	if err := UpdateUserConsumer(ctx, db, consumer); err != nil {
		return err
	}
	return nil
}

// ConsumerRemoveGroup removes given group from all consumers that using it, set warning and disabled state if needed.
func ConsumerRemoveGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, g *sdk.Group) error {
	// Load all consumers that refer to the group
	cs, err := LoadUserConsumersByGroupID(ctx, db, g.ID)
	if err != nil {
		return err
	}

	for i := range cs {
		// Remove the group id from the consumer and add a warning to the consumer
		if !cs[i].AuthConsumerUser.GroupIDs.Contains(g.ID) && !cs[i].AuthConsumerUser.InvalidGroupIDs.Contains(g.ID) {
			continue
		}

		cs[i].AuthConsumerUser.GroupIDs.Remove(g.ID)
		cs[i].AuthConsumerUser.InvalidGroupIDs.Remove(g.ID)

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
		if len(cs[i].AuthConsumerUser.GroupIDs) == 0 && !cs[i].Disabled {
			cs[i].Disabled = true
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningLastGroupRemoved())
		}

		if err := UpdateUserConsumer(ctx, db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerInvalidateGroupForUser set group as invalid in all user's consumers and set warning.
func ConsumerInvalidateGroupForUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, g *sdk.Group, u *sdk.AuthentifiedUser) error {
	// If an admin is removed from a group we want to preserve its consumer with this group
	if u.Ring == sdk.UserRingAdmin {
		return nil
	}

	// Load all consumers for the user
	cs, err := LoadUserConsumersByUserID(ctx, db, u.ID)
	if err != nil {
		return err
	}
	for i := range cs {
		if len(cs[i].AuthConsumerUser.GroupIDs) == 0 || !cs[i].AuthConsumerUser.GroupIDs.Contains(g.ID) {
			continue
		}

		// Remove the group id from slice and add it to the invalid ones
		cs[i].AuthConsumerUser.GroupIDs.Remove(g.ID)
		cs[i].AuthConsumerUser.InvalidGroupIDs = append(cs[i].AuthConsumerUser.InvalidGroupIDs, g.ID)
		cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningGroupInvalid(g.ID, g.Name))

		// If there is no group left in the consumer we want to disable it
		if len(cs[i].AuthConsumerUser.GroupIDs) == 0 {
			cs[i].Disabled = true
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningLastGroupRemoved())
		}

		if err := UpdateUserConsumer(ctx, db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerRestoreInvalidatedGroupForUser checks if there are consumers for given user where the group was invalidated, then
// restore it and remove warning.
func ConsumerRestoreInvalidatedGroupForUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, groupID int64, userID string) error {
	// Load all consumers for the user
	cs, err := LoadUserConsumersByUserID(ctx, db, userID)
	if err != nil {
		return err
	}
	for i := range cs {
		if len(cs[i].AuthConsumerUser.InvalidGroupIDs) == 0 || !cs[i].AuthConsumerUser.InvalidGroupIDs.Contains(groupID) {
			continue
		}

		// Remove the group id from slice and add it to the valid ones
		cs[i].AuthConsumerUser.InvalidGroupIDs.Remove(groupID)
		cs[i].AuthConsumerUser.GroupIDs = append(cs[i].AuthConsumerUser.GroupIDs, groupID)

		// If the consumer was disabled because there was no group left inside, it can be re-enable
		cs[i].Disabled = false

		// Clean warnings, removes warning for current group and last group removed warning if exists
		filteredWarnings := make(sdk.AuthConsumerWarnings, 0, len(cs[i].Warnings))
		for _, w := range cs[i].Warnings {
			if (w.Type == sdk.WarningGroupInvalid && w.GroupID != groupID) ||
				w.Type == sdk.WarningGroupRemoved {
				filteredWarnings = append(filteredWarnings, w)
			}
		}
		cs[i].Warnings = filteredWarnings

		if err := UpdateUserConsumer(ctx, db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerInvalidateGroupsForUser set groups as invalid if the user is not a member in all user's consumers and set warning.
func ConsumerInvalidateGroupsForUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID string, userGroupIDs sdk.Int64Slice) error {
	// Load all consumers for the user
	cs, err := LoadUserConsumersByUserID(ctx, db, userID, LoadUserConsumerOptions.WithConsumerGroups)
	if err != nil {
		return err
	}
	for i := range cs {
		// If there is no group in the consumer we can skip it
		if len(cs[i].AuthConsumerUser.GroupIDs) == 0 {
			continue
		}

		for j := range cs[i].AuthConsumerUser.Groups {
			if userGroupIDs.Contains(cs[i].AuthConsumerUser.Groups[j].ID) {
				continue
			}

			// Remove the group id from slice and add it to the invalid ones
			cs[i].AuthConsumerUser.GroupIDs.Remove(cs[i].AuthConsumerUser.Groups[j].ID)
			cs[i].AuthConsumerUser.InvalidGroupIDs = append(cs[i].AuthConsumerUser.InvalidGroupIDs, cs[i].AuthConsumerUser.Groups[j].ID)
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningGroupInvalid(cs[i].AuthConsumerUser.Groups[j].ID, cs[i].AuthConsumerUser.Groups[j].Name))
		}

		// If there is no group left in the consumer we want to disable it
		if len(cs[i].AuthConsumerUser.GroupIDs) == 0 {
			cs[i].Disabled = true
			cs[i].Warnings = append(cs[i].Warnings, sdk.NewConsumerWarningLastGroupRemoved())
		}

		if err := UpdateUserConsumer(ctx, db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}

// ConsumerRestoreInvalidatedGroupsForUser restore invalidated group for all user's consumer, this should be used only for a admin user.
func ConsumerRestoreInvalidatedGroupsForUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID string) error {
	// Load all consumers for the user
	cs, err := LoadUserConsumersByUserID(ctx, db, userID)
	if err != nil {
		return err
	}
	for i := range cs {
		if len(cs[i].AuthConsumerUser.InvalidGroupIDs) == 0 {
			continue
		}

		// Moves invalid group ids to valid slice
		cs[i].AuthConsumerUser.GroupIDs = append(cs[i].AuthConsumerUser.GroupIDs, cs[i].AuthConsumerUser.InvalidGroupIDs...)
		cs[i].AuthConsumerUser.InvalidGroupIDs = nil

		// If the consumer was disabled because there was no group left inside, it can be re-enable
		cs[i].Disabled = false

		// Clean warnings, removes warning for invalid groups and last group removed warning if exists
		filteredWarnings := make(sdk.AuthConsumerWarnings, 0, len(cs[i].Warnings))
		for _, w := range cs[i].Warnings {
			if w.Type == sdk.WarningGroupRemoved {
				filteredWarnings = append(filteredWarnings, w)
			}
		}
		cs[i].Warnings = filteredWarnings

		if err := UpdateUserConsumer(ctx, db, &cs[i]); err != nil {
			return err
		}
	}

	return nil
}
