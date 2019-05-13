package api

/*
func (api *API) importUsersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var users = []sdk.User{}
		if err := service.UnmarshalBody(r, &users); err != nil {
			return err
		}

		_, hashedToken, err := user.GeneratePassword()
		if err != nil {
			return sdk.WrapError(err, "Error while generate Token Verify for new user")
		}

		errors := map[string]string{}
		for _, u := range users {
			if err := user.InsertUser(api.mustDB(), &u, &sdk.Auth{
				EmailVerified:  true,
				DateReset:      0,
				HashedPassword: hashedToken,
			}); err != nil {
				oldU, err := user.LoadUserWithoutAuth(api.mustDB(), u.Username)
				if err != nil {
					errors[u.Username] = err.Error()
					continue
				}
				u.ID = oldU.ID
				u.Auth = sdk.Auth{
					EmailVerified: true,
					DateReset:     0,
				}
				if err := user.UpdateUserAndAuth(api.mustDB(), u); err != nil {
					errors[u.Username] = err.Error()
				}
			}
		}

		return service.WriteJSON(w, errors, http.StatusOK)
	}
}
*/
