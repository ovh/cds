package user

/*
func GetDeprecatedUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadDeprecatedUserOptionFunc) ([]sdk.User, error) {
	dus := []DeprecatedUser{}

	if err := gorpmapping.GetAll(ctx, db, q, &dus); err != nil {
		return nil, sdk.WrapError(err, "cannot get deprecated users")
	}

	pus := make([]*sdk.User, len(dus))
	for i := range dus {
		pus[i] = &dus[i].Data
		pus[i].ID = dus[i].ID
		pus[i].Admin = dus[i].Admin
		pus[i].Origin = dus[i].Origin
	}

	if len(pus) > 0 {
		for i := range opts {
			if err := opts[i](db, pus...); err != nil {
				return nil, err
			}
		}
	}

	us := make([]sdk.User, len(pus))
	for i := range pus {
		us[i] = *pus[i]
	}

	return us, nil
}

func GetDeprecatedUser(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...LoadDeprecatedUserOptionFunc) (*sdk.User, error) {
	var du DeprecatedUser

	found, err := gorpmapping.Get(ctx, db, q, &du)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get deprecated user")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	pu := &du.Data
	pu.ID = du.ID
	pu.Admin = du.Admin
	pu.Origin = du.Origin

	for i := range opts {
		if err := opts[i](db, pu); err != nil {
			return nil, err
		}
	}

	return pu, nil
}

// LoadDeprecatedUsersWithoutAuthByIDs returns deprecated users from database for given ids.
func LoadDeprecatedUsersWithoutAuthByIDs(ctx context.Context, db gorp.SqlExecutor, ids []int64, opts ...LoadDeprecatedUserOptionFunc) (sdk.Users, error) {
	query := gorpmapping.NewQuery(`
    SELECT id, username, admin, data, origin
    FROM "user"
    WHERE id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(ids))
	return GetDeprecatedUsers(ctx, db, query, opts...)
}

// LoadDeprecatedUserWithoutAuthByID returns deprecated user from database for given id.
func LoadDeprecatedUserWithoutAuthByID(ctx context.Context, db gorp.SqlExecutor, id int64, opts ...LoadDeprecatedUserOptionFunc) (*sdk.User, error) {
	query := gorpmapping.NewQuery(`
    SELECT id, username, admin, data, origin
    FROM "user"
    WHERE id = $1
  `).Args(id)
	return GetDeprecatedUser(ctx, db, query, opts...)
}

func insertDeprecatedUser(db gorp.SqlExecutor, u *sdk.User) error {
	su, err := json.Marshal(u)
	if err != nil {
		return sdk.WithStack(err)
	}
	query := `INSERT INTO "user" (username, admin, data, created, origin) VALUES($1,$2,$3,$4,$5) RETURNING id`
	if err := db.QueryRow(query, u.Username, u.Admin, su, time.Now(), u.Origin).Scan(&u.ID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// DeleteUserWithDependencies Delete user and all his dependencies
func DeleteUserWithDependencies(db gorp.SqlExecutor, u *sdk.User) error {
	if err := deleteUserFromUserGroup(db, u); err != nil {
		return sdk.WrapError(err, "User cannot be removed from group_user table")
	}

	if err := deleteUser(db, u); err != nil {
		return sdk.WrapError(err, "User cannot be removed from user table")
	}
	return nil
}

func deleteUserFromUserGroup(db gorp.SqlExecutor, u *sdk.User) error {
	query := `DELETE FROM "group_user" WHERE user_id=$1`
	_, err := db.Exec(query, u.ID)
	return sdk.WithStack(err)
}

func deleteUser(db gorp.SqlExecutor, u *sdk.User) error {
	query := `DELETE FROM "user" WHERE id=$1`
	_, err := db.Exec(query, u.ID)
	return sdk.WithStack(err)
}
*/
