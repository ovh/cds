package main

import (
	"testing"

	"github.com/ovh/tat"
	"github.com/ovh/tat/api/tests"
	"github.com/stretchr/testify/assert"
)

var topicsController = &TopicsController{}

func TestTopicCreateListAndDelete(t *testing.T) {
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

	topic, err := client.TopicCreate(tat.TopicCreateJSON{
		Topic:       "/" + tests.RandomString(t, 10),
		Description: "this is a test",
	})

	assert.NotNil(t, topic)
	assert.NoError(t, err)
	if topic == nil {
		t.Fail()
		return
	}
	t.Logf("Topic %s created", topic.Topic)
	assert.NotZero(t, topic.ID)

	topics, err := client.TopicList(nil)
	assert.NotNil(t, topics)
	assert.NoError(t, err)

	t.Log("Delete all topics")
	for _, to := range topics.Topics {
		_, err := client.TopicDelete(tat.TopicNameJSON{Topic: to.Topic})
		assert.NoError(t, err)
	}

}

func TestTruncateAndDeleteAllTopics(t *testing.T) {
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

	topics, err := client.TopicList(nil)
	assert.NotNil(t, topics)
	assert.NoError(t, err)

	t.Log("Delete all topics")
	for _, to := range topics.Topics {
		_, err := client.TopicTruncate(tat.TopicNameJSON{Topic: to.Topic})
		assert.NoError(t, err)
		_, err = client.TopicDelete(tat.TopicNameJSON{Topic: to.Topic})
		assert.NoError(t, err)
	}
}

func TestListTopicsFromCache(t *testing.T) {
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

	topic, err := client.TopicCreate(tat.TopicCreateJSON{
		Topic:       "/" + tests.RandomString(t, 10),
		Description: "this is a test",
	})

	assert.NotNil(t, topic)
	assert.NoError(t, err)
	if topic == nil {
		t.Fail()
		return
	}
	t.Logf("Topic %s created", topic.Topic)

	topics, err := client.TopicList(nil)
	assert.NotNil(t, topics)
	assert.NoError(t, err)

	assert.Equal(t, 1, topics.Count)
	assert.Equal(t, 1, len(topics.Topics))

	_, err = client.TopicCreate(tat.TopicCreateJSON{
		Topic:       "/" + tests.RandomString(t, 10),
		Description: "this is a test",
	})
	assert.NoError(t, err)

	_, err = client.TopicCreate(tat.TopicCreateJSON{
		Topic:       "/" + tests.RandomString(t, 10),
		Description: "this is a test",
	})
	assert.NoError(t, err)

	topics, err = client.TopicList(nil)
	assert.NotNil(t, topics)
	assert.NoError(t, err)

	assert.Equal(t, 3, topics.Count)
	assert.Equal(t, 3, len(topics.Topics))

	_, err = client.TopicDelete(tat.TopicNameJSON{Topic: topic.Topic})
	assert.NoError(t, err)
}

func TestTopicsFilterManage(t *testing.T) {
	tests.Init(t)
	router := tests.Router(t)
	client := tests.TATClient(t, tests.AdminUser)

	initRoutesTopics(router, tests.FakeAuthHandler(t, tests.AdminUser, "X-TAT-TEST", true, false))

	topic, err := client.TopicCreate(tat.TopicCreateJSON{
		Topic:       "/" + tests.RandomString(t, 10),
		Description: "this is a test",
	})

	assert.NotNil(t, topic)
	assert.NoError(t, err)
	if topic == nil {
		t.Fail()
		return
	}
	t.Logf("Topic %s created", topic.Topic)

	_, err = client.TopicAddFilter(tat.Filter{
		Topic:    topic.Topic,
		Title:    "myTitle",
		Criteria: tat.FilterCriteria{Label: "fooLabel"},
		Hooks: []tat.Hook{
			{Type: tat.HookTypeWebHook, Destination: "fooDestination", Action: tat.MessageActionCreate},
		}})
	t.Logf("Filter created on %s", topic.Topic)

	assert.NoError(t, err)

	created, err := client.TopicOne(topic.Topic)
	assert.NotNil(t, created)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(created.Topic.Filters), "topic should have one filter")
	t.Logf("Filter on %s: %+v", topic.Topic, created.Topic.Filters)

	out, erru := client.TopicUpdateFilter(tat.Filter{
		Topic:    topic.Topic,
		ID:       created.Topic.Filters[0].ID,
		Title:    "myTitleUpdate",
		Criteria: tat.FilterCriteria{Tag: "fooTag"},
		Hooks: []tat.Hook{
			{Type: tat.HookTypeWebHook, Destination: "fooDestination", Enabled: true, Action: tat.MessageActionCreate},
		}})
	assert.NoError(t, erru)
	t.Logf("out after update filter on %s: %s", topic.Topic, string(out))

	createdUpdated, errc := client.TopicOne(topic.Topic)
	assert.NotNil(t, createdUpdated)
	assert.NoError(t, errc)
	assert.Equal(t, 1, len(createdUpdated.Topic.Filters), "topic should have one filter")
	assert.Equal(t, "fooTag", createdUpdated.Topic.Filters[0].Criteria.Tag, "criteria tag should be fooTag")
	assert.Equal(t, "", createdUpdated.Topic.Filters[0].Criteria.Label, "criteria label should be empty")
	assert.Equal(t, true, createdUpdated.Topic.Filters[0].Hooks[0].Enabled, "hook should be enabled")
	t.Logf("Filter on %s: %+v", topic.Topic, createdUpdated.Topic.Filters)

	_, err = client.TopicRemoveFilter(tat.Filter{Topic: topic.Topic, ID: createdUpdated.Topic.Filters[0].ID})
	assert.NoError(t, err)

	createdUpdatedBis, errc := client.TopicOne(topic.Topic)
	assert.NotNil(t, createdUpdatedBis)
	assert.NoError(t, errc)
	assert.Equal(t, 0, len(createdUpdatedBis.Topic.Filters), "topic should have no filter")

	_, err = client.TopicDelete(tat.TopicNameJSON{Topic: created.Topic.Topic})
	assert.NoError(t, err)
}
