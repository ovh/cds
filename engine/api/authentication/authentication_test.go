package authentication_test

/*func Test_verifyToken(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	_, jwt, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)

	test.NoError(t, err)
	t.Logf("jwt token: %s", jwt)

	_, err = authentication.VerifyToken(jwt)
	test.NoError(t, err)

	_, err = authentication.VerifyToken("this is not a jwt token")
	assert.Error(t, err)
}

func TestIsValid(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(1 * time.Second)
	token, jwtToken, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	test.NoError(t, authentication.Insert(db, &token))
	_, isValid, err := authentication.IsValid(db, jwtToken)
	test.NoError(t, err)
	assert.True(t, isValid)

	grp2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))
	token.Groups = append(token.Groups, *grp2)
	jwtToken2, err := authentication.Regen(&token)
	test.NoError(t, err)

	_, isValid, err = authentication.IsValid(db, jwtToken2)
	test.NoError(t, err)
	assert.False(t, isValid)

	// Wait for expiration, the token should be now expired
	time.Sleep(2 * time.Second)
	_, isValid, err = authentication.IsValid(db, jwtToken)
	assert.Error(t, err)
	assert.False(t, isValid)
}

func TestXSRFToken(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	usr1, _ := assets.InsertLambdaUser(db)
	grp1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	exp := time.Now().Add(5 * time.Minute)
	token, _, err := authentication.New(*usr1, []sdk.Group{*grp1}, []string{sdk.AccessTokenScopeALL}, "cds_test", "cds test", exp)
	test.NoError(t, err)

	x := authentication.StoreXSRFToken(cache, token)
	isValid := authentication.CheckXSRFToken(cache, token, x)
	assert.True(t, isValid)

	isValid = authentication.CheckXSRFToken(cache, token, sdk.UUID())
	assert.False(t, isValid)
}*/
