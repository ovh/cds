package main

import (
	"fmt"
	"testing"

	"github.com/ovh/tat"
	"github.com/ovh/tat/api/tests"
	"github.com/stretchr/testify/assert"
)

var usersController = &UsersController{}

// TestUserMe tests non-admin user, authenticated on tat
// GET on /user/me, check HTTP 200
func TestUserMe(t *testing.T) {
	tests.Init(t)

	router := tests.Router(t)

	initRoutesGroups(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesMessages(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesPresences(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesTopics(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesUsers(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesStats(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesSystem(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))

	client := tests.TATClient(t, "")
	u, err := client.UserMe()
	assert.NotNil(t, u)
	assert.NoError(t, err)

}

func TestCreateUser(t *testing.T) {
	tests.Init(t)

	router := tests.Router(t)

	initRoutesGroups(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesMessages(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesPresences(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesTopics(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesUsers(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesStats(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesSystem(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))

	client := tests.TATClient(t, "")

	username := "tat.integration.test.users." + tests.RandomString(t, 10)
	r, err := client.UserAdd(tat.UserCreateJSON{
		Username: username,
		Fullname: fmt.Sprintf("User %s created for Tat Integration Test", username),
		Email:    fmt.Sprintf("%s@tat.foo", username),
	})

	assert.NotNil(t, r)
	assert.NoError(t, err)
	t.Logf("User created, return from tat:%s", r)
}
