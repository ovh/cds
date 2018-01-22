package main

import (
	"fmt"
	"testing"

	"github.com/ovh/tat"
	"github.com/ovh/tat/api/tests"
	"github.com/stretchr/testify/assert"
	"log"
)

var messagesCtrl = &MessagesController{}

func TestMessagesList(t *testing.T) {
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
	if topic != nil {
		t.Logf("Topic %s created", topic.Topic)
	}

	defer client.TopicDelete(tat.TopicNameJSON{Topic: topic.Topic})
	defer client.TopicTruncate(tat.TopicNameJSON{Topic: topic.Topic})

	_, err = client.TopicParameter(tat.TopicParameters{
		Topic:                topic.Topic,
		Recursive:            false,
		CanDeleteMsg:         true,
		AdminCanDeleteAllMsg: true,
	})
	assert.NoError(t, err)

	message, err := client.MessageAdd(tat.MessageJSON{
		Text:  "test test",
		Topic: topic.Topic,
	})
	assert.NotNil(t, message)
	assert.NoError(t, err)
	if topic != nil {
		t.Log(message.Info)
	}

	messages, err := client.MessageList(topic.Topic, nil)
	assert.NotNil(t, topic)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(messages.Messages))

	messages, err = client.MessageList(topic.Topic, nil)
	assert.NotNil(t, topic)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(messages.Messages))

	message, err = client.MessageAdd(tat.MessageJSON{
		Text:   "#test2 #test2",
		Topic:  topic.Topic,
		Labels: []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelB", Color: "#eeeeee"}},
	})
	assert.NotNil(t, message)
	assert.NoError(t, err)
	if topic != nil {
		t.Log(message.Message)
	}

	message, err = client.MessageRelabel(topic.Topic, message.Message.ID, []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelD", Color: "#eeeeee"}}, []string{"labelA"})
	assert.NotNil(t, message)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(message.Message.Labels), "this message should have 3 labels")

	message, err = client.MessageRelabel(topic.Topic, message.Message.ID, []tat.Label{{Text: "labelF", Color: "#eeeeee"}}, nil)
	assert.NotNil(t, message)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(message.Message.Labels), "this message should have 1 label")

	messages, err = client.MessageList(topic.Topic, nil)
	assert.NotNil(t, topic)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(messages.Messages))
	if len(messages.Messages) != 2 {
		t.Fail()
		return
	}

	messagesSearch, err := client.MessageList(topic.Topic, &tat.MessageCriteria{Text: "#test2"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(messagesSearch.Messages))

	if messages.Messages[0].DateCreation < messages.Messages[1].DateCreation {
		t.Log("Wrong order")
		t.Fail()
	}

	_, err = client.MessageDelete(message.Message.ID, topic.Topic, false, false)
	assert.NoError(t, err)

	messages, err = client.MessageList(topic.Topic, nil)
	assert.NotNil(t, topic)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(messages.Messages))

	msgs := []tat.MessageJSON{{Text: "MessageA #createBulk", Topic: topic.Topic}, {Text: "MessageB #createBulk", Topic: topic.Topic}}
	messagesJSON, err := client.MessageAddBulk(msgs)
	assert.NotNil(t, messagesJSON)
	assert.NoError(t, err)

	messagesSearchBulk, err := client.MessageList(topic.Topic, &tat.MessageCriteria{Text: "#createBulk"})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(messagesSearchBulk.Messages))

	message, err = client.MessageAdd(tat.MessageJSON{
		Text:   "foo #tag:aaa",
		Topic:  topic.Topic,
		Labels: []tat.Label{{Text: "xxx:yyy", Color: "#eeeeee"}},
	})
	assert.NoError(t, err)

	message, err = client.MessageAdd(tat.MessageJSON{
		Text:   "foo #tag:bbb",
		Topic:  topic.Topic,
		Labels: []tat.Label{{Text: "xxx:zzz", Color: "#eeeeee"}},
	})
	assert.NoError(t, err)

	messagesSearchStartLabel, err := client.MessageList(topic.Topic, &tat.MessageCriteria{StartLabel: "xxx"})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(messagesSearchStartLabel.Messages))

	messagesSearchStartTag, err := client.MessageList(topic.Topic, &tat.MessageCriteria{StartTag: "tag:aa"})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(messagesSearchStartTag.Messages))

	messages, err = client.MessageList(topic.Topic, &tat.MessageCriteria{SortBy: "-dateUpdate"})
	assert.NoError(t, err)
	assert.Equal(t, 5, len(messages.Messages))

	// Need to change ErrorLogFunc because next test will imply logError
	// and we do not want to fail because it is logical
	tat.ErrorLogFunc = log.Printf
	messages, err = client.MessageList(topic.Topic, &tat.MessageCriteria{SortBy: "-dateUpdate", TreeView: tat.TreeViewOneTree})
	assert.Nil(t, messages)
	//assert.NoError(t, err)
	assert.EqualError(t, err, "Response code:403 (want:200) with Body:{\"error\":\"Sort must be -dateCreation or treeView will not work\"}\n{\"messages\":[],\"isTopicRw\":true,\"isTopicAdmin\":false}\n")

	msg := tat.MessageJSON{
		Topic:        topic.Topic,
		TagReference: "foocreate",
		Labels:       []tat.Label{{Text: "labelA", Color: "#eeeeee"}, {Text: "labelD", Color: "#eeeeee"}},
		Text:         "new text",
	}
	message, err = client.MessageRelabelOrCreate(msg)
	assert.NotNil(t, message)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(message.Message.Labels), "this message should have 2 labels")

	tat.ErrorLogFunc = t.Errorf
}

func TestMessagesInsert(t *testing.T) {
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
	if topic != nil {
		t.Logf("Topic %s created", topic.Topic)
	}

	defer client.TopicDelete(tat.TopicNameJSON{Topic: topic.Topic})
	defer client.TopicTruncate(tat.TopicNameJSON{Topic: topic.Topic})

	message, err := client.MessageAdd(tat.MessageJSON{Text: "test test", Topic: topic.Topic})
	assert.NotNil(t, message)
	assert.NoError(t, err)

	for nb := 0; nb < 35; nb++ {
		r, errr := client.MessageAdd(tat.MessageJSON{Text: fmt.Sprintf("reply %d", nb), IDReference: message.Message.ID, Topic: topic.Topic})
		t.Logf("Reply %d %s added on root %s", nb, r.Message.ID, message.Message.ID)
		assert.NotNil(t, r)
		assert.NoError(t, errr)
	}

	replies, errl := client.MessageList(topic.Topic, &tat.MessageCriteria{IDMessage: message.Message.ID, TreeView: tat.TreeViewOneTree})
	assert.NoError(t, errl)
	assert.Equal(t, 1, len(replies.Messages))
	assert.Equal(t, 30, len(replies.Messages[0].Replies))

}
