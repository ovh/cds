package main

import (
	"testing"

	"github.com/ovh/tat"
	"github.com/ovh/tat/api/tests"
	"github.com/stretchr/testify/assert"
)

var groupsController = GroupsController{}

func TestAddAndDeleteGroup(t *testing.T) {
	tests.Init(t)
	router := tests.Router(t)
	client := tests.TATClient(t, tests.AdminUser)

	initRoutesGroups(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesMessages(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesPresences(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesTopics(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesUsers(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesStats(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))
	initRoutesSystem(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))

	group, err := client.GroupCreate(tat.GroupJSON{
		Description: "Group admin for tests",
		Name:        tests.RandomString(t, 10),
	})
	assert.NoError(t, err)

	err = client.GroupDelete(group.Name)
	assert.NoError(t, err)
}
