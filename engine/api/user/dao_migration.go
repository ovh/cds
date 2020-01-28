package user

/*
func getUserMigrations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]MigrationUser, error) {
	ms := []migrationUser{}

	if err := gorpmapping.GetAll(ctx, db, q, &ms); err != nil {
		return nil, sdk.WrapError(err, "cannot get user migrations")
	}

	// Check signature of data, if invalid do not return it
	verifiedUserMigrations := make([]MigrationUser, 0, len(ms))
	for i := range ms {
		isValid, err := gorpmapping.CheckSignature(ms[i], ms[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "user.getUserMigrations> user migration for authentified user %s and user %d data corrupted", ms[i].AuthentifiedUserID, ms[i].UserID)
			continue
		}
		verifiedUserMigrations = append(verifiedUserMigrations, ms[i].MigrationUser)
	}

	return verifiedUserMigrations, nil
}

// LoadMigrationUsersByUserIDs returns all authentified user migration entries for given user ids.
func LoadMigrationUsersByUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []string) (MigrationUsers, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_migration
    WHERE authentified_user_id = ANY(string_to_array($1, ',')::text[])
  `).Args(gorpmapping.IDStringsToQueryString(userIDs))
	return getUserMigrations(ctx, db, query)
}

// LoadMigrationUsersByDeprecatedUserIDs returns all authentified user migration entries for given deprecated user ids.
func LoadMigrationUsersByDeprecatedUserIDs(ctx context.Context, db gorp.SqlExecutor, deprecatedUserIDs []int64) (MigrationUsers, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_migration
    WHERE user_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(deprecatedUserIDs))
	return getUserMigrations(ctx, db, query)
}

func insertUserMigration(ctx context.Context, db gorp.SqlExecutor, m *MigrationUser) error {
	mi := migrationUser{MigrationUser: *m}
	if err := gorpmapping.InsertAndSign(ctx, db, &mi); err != nil {
		return sdk.WrapError(err, "unable to insert into table authentified_user_migration")
	}
	*m = mi.MigrationUser
	return nil
}
*/
