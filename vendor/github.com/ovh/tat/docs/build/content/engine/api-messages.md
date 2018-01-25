---
title: "API - Messages"
weight: 2
toc: true
prev: "/engine/general"
next: "/engine/api-topics"

---

## Store a new message

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "text": "text" }' \
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

You can add labels from the creation

```bash
curl -XPOST \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "text": "text", "dateCreation": 11123232, "labels": [{"text": "labelA", "color": "#eeeeee"}, {"text": "labelB", "color": "#ffffff"}] }' \
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

You can add replies from the creation

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
  -d '{ "text": "text", "replies":["reply A", "reply B"] }' \
  https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

You can add replies, with labels, from the creation

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
  -d '{ "text": "text msg root", "messages": [{ "text": "text reply", "labels": [{"text": "labelA", "color": "#eeeeee"}] }] }' \
  https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

If you use a `system user`, you can force message's date

```bash
curl -XPOST \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "text": "text", "dateCreation": 11123232 }' \
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

Return HTTP 201 if OK

## Store some messages

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '[{ "text": "text message A" },{ "text": "text message B", "labels": [{"text": "labelA", "color": "#eeeeee"}] }]' \
	https://<tatHostname>:<tatPort>/messages/a-topic/sub-topic
```

## Action on a existing message

Reply, Like, Unlike, Add Label, Remove Label, etc... use idReference but it's possible to use :

* TagReference
* StartTagReference
* LabelReference
* StartLabelReference
* OnlyRootReference (true by default)

```bash
curl -XPOST \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "text": "text", "startTagReference": "keyTag:", "action": "reply"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

If several messages match to your request, Tat gives you a HTTP Bad Request.

## Reply to a message

```bash
curl -XPOST \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "text": "text", "idReference": "9797q87KJhqsfO7Usdqd", "action": "reply"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Reply to a message, with create root message if necessary

This request will :

* Create the root message with text `the root message #aaa`
* If a message with a tag `#aaa` already exists, this message will be used to add replies on it
* Add two replies `reply A` and `reply B`

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
  -d '{ "text": "the root message #aaa", "replies":["reply A", "reply B"], "tagReference": "aaa"}' \
  https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Like a message
```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "like"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Unlike a message
```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "unlike"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Add a label to a message
*option* is the background color of the label.

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "label", "text": "myLabel", "option": "rgba(143,199,148,0.61)"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Remove a label from a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "unlabel", "text": "myLabel"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Remove all labels and add new ones

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "relabel", "labels": [{"text": "labelA", "color": "#eeeeee"}, {"text": "labelB", "color": "#ffffff"}]}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

Return HTTP 201 if OK

## Remove some labels and add new ones

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "relabel", "labels": [{"text": "labelA", "color": "#eeeeee"}, {"text": "labelB", "color": "#ffffff"}], "options": ["labelAToRemove", "labelAToRemove"] }'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Remove all labels and add new ones on existing message, create message otherwise

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "tagReference": "foo", "text": "a text with #foo" "action": "relabelorcreate", "labels": [{"text": "labelA", "color": "#eeeeee"}, {"text": "labelB", "color": "#ffffff"}]}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

If a message with tag `foo` already exists on topic, apply new labels. If message does not exist, a new message will be created.

## Update a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "tagReference": "aaa", "onlyRootReference": "false", "action": "update", "text": "my New Mesage updated"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Concat a message : adding additional text to one message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "concat", "text": " additional text"}'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Move a message to another topic

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "move", "option": "/newTopic/subNewTopic"}'\
	https://<tatHostname>:<tatPort>/message/oldTOpic/oldSubTopic
```

## Delete a message
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/message/nocascade/9797q87KJhqsfO7Usdqd/topic/subTopic
```

## Delete a message and its replies
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/message/cascade/9797q87KJhqsfO7Usdqd/topic/subTopic
```

## Delete a message and its replies, even if it's in Tasks Topic of one user
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/message/cascadeforce/9797q87KJhqsfO7Usdqd/topic/subTopic
```

## Delete a list of messages
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/messages/nocascade/topic/subTopic?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name
```

see https://github.com/ovh/tat#parameters for all parameters

## Delete a list of messages and its replies
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/messages/cascade/topic/subTopic?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name
```

see https://github.com/ovh/tat#parameters for all parameters

## Delete a list of messages and its replies, even if it's a reply or it's in Tasks Topic of one user
```bash
curl -XDELETE \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/messages/cascadeforce/topic/subTopic?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name
```

see https://github.com/ovh/tat#parameters for all parameters

## Create a task from a message
Add a message to topic: `/Private/username/Tasks`.

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "task" }'\
	https://<tatHostname>:<tatPort>/message/Private/username/Tasks
```

## Remove a message from tasks
Remove a message from the topic: /Private/username/Tasks

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "untask" }'\
	https://<tatHostname>:<tatPort>/message/Private/username/Tasks
```

## Vote UP a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "voteup" }'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Remove a Vote UP from a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "unvoteup" }'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Vote Down a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "votedown" }'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Remove Vote Down from a message

```bash
curl -XPUT \
    -H 'Content-Type: application/json' \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "idReference": "9797q87KJhqsfO7Usdqd", "action": "unvotedown" }'\
	https://<tatHostname>:<tatPort>/message/a-topic/sub-topic
```

## Getting Messages List
```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/<topic>?skip=<skip>&limit=<limit>
curl -XGET https://<tatHostname>:<tatPort>/messages/<topic>?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name
```

Getting messages on one Public Topic (Read Only):

```bash
curl -XGET https://<tatHostname>:<tatPort>/read/<topic>?skip=<skip>&limit=<limit>
curl -XGET https://<tatHostname>:<tatPort>/read/<topic>?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name
```

### Parameters

* `allIDMessage`          Search in All ID Message (idMessage, idReply, idRoot)
* `andLabel`              Search by label (and) : could be labelA,labelB
* `andTag`                Search by tag (and) : could be tagA,tagB
* `idMessage`             Search by IDMessage
* `inReplyOfID`           Search by IDMessage InReply
* `inReplyOfIDRoot`       Search by IDMessage IdRoot
* `label`                 Search by label: could be labelA,labelB
* `dateMaxCreation`       Search by dateCreation (timestamp), select messages where dateCreation <= dateMaxCreation
* `dateMaxUpdate`         Search by dateUpdate (timestamp), select messages where dateUpdate <= dateMaxUpdate
* `dateMinCreation`       Search by dateCreation (timestamp), select messages where dateCreation >= dateMinCreation
* `dateMinUpdate`         Search by dateUpdate (timestamp), select messages where dateUpdate >= dateMinUpdate
* `dateRefCreation`            This have to be used with dateRefDeltaMinCreation and / or dateRefDeltaMaxCreation. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear
* `dateRefDeltaMaxCreation`    Add seconds to dateRefCreation flag
* `dateRefDeltaMaxUpdate`      Add seconds to dateRefUpdate flag
* `dateRefDeltaMinCreation`    Add seconds to dateRefCreation flag
* `dateRefDeltaMinUpdate`      Add seconds to dateRefUpdate flag
* `dateRefUpdate`              This have to be used with dateRefDeltaMinUpdate and / or dateRefDeltaMaxUpdate. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear
* `lastHourMaxCreation`   Search by dateCreation, select messages where dateCreation <= Now Beginning Of Hour - (60 * lastHourMaxCreation)
* `lastHourMaxUpdate`     Search by dateUpdate, select messages where dateUpdate <= Now Beginning Of Hour - (60 * lastHourMaxCreation)
* `lastHourMinCreation`   Search by dateCreation, select messages where dateCreation >= Now Beginning Of Hour - (60 * lastHourMinCreation)
* `lastHourMinUpdate`     Search by dateUpdate, select messages where dateUpdate >= Now Beginning Of Hour - (60 * lastHourMinCreation)
* `lastMaxCreation`       Search by dateCreation (duration in second), select messages where dateCreation <= now - lastMaxCreation
* `lastMaxCreation`       Search by dateCreation (duration in second), select messages where dateCreation <= now - lastMaxCreation
* `lastMaxUpdate`         Search by dateUpdate (duration in second), select messages where dateUpdate <= now - lastMaxCreation
* `lastMinCreation`       Search by dateCreation (duration in second), select messages where dateCreation >= now - lastMinCreation
* `lastMinUpdate`         Search by dateUpdate (duration in second), select messages where dateUpdate >= now - lastMinCreation
* `limitMaxNbReplies`     In onetree mode, filter root messages with min or equals maxNbReplies
* `limitMaxNbVotesDown`   Search by nbVotesDown
* `limitMaxNbVotesUP`     Search by nbVotesUP
* `limitMinNbReplies`     In onetree mode, filter root messages with more or equals minNbReplies
* `limitMinNbVotesDown`   Search by nbVotesDown
* `limitMinNbVotesUP`     Search by nbVotesUP
* `notLabel`              Search by label (exclude): could be labelA,labelB
* `notTag`                Search by tag (exclude) : could be tagA,tagB
* `onlyCount`             onlyCount=true: only count messages, without retrieve msg. limit, skip, treeview criterias are ignored.
onlyMsgRoot string           onlyMsgRoot=true: restricts to root message only (inReplyOfIDRoot empty). If treeView is used, limit search criteria to root * `message` are still given, independently of search criteria.
* `startLabel`            Search by a label prefix: startLabel='mykey:,myKey2:'
* `startTag`              Search by a tag prefix: startTag='mykey:,myKey2:'
* `tag`                   Search by tag : could be tagA,tagB
* `text`                  Search by text
* `topic`                 Search by topic
* `treeView`              Tree View of messages: onetree or fulltree. Default: notree
* `username`              Search by username : could be usernameA,usernameB
* `sortBy`                Sort message. Use '-' to reverse sort. Default is --sortBy=-dateCreation. You can use: text, topic, inReplyOfID, inReplyOfIDRoot, nbLikes, labels, likers, votersUP, votersDown, nbVotesUP, nbVotesDown, userMentions, urls, tags, dateCreation, dateUpdate, author, nbReplies

### Examples

#### GET 100 last created messages messages
```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA?skip=0&limit=100
```

#### Filter by date Creation

This will return 100 messages created between 16/7/2014, 22:55:46 and 16/8/2014, 22:55:46

* 1405544146 is 16/7/2014, 22:55:46
* 1408222546 is 16/8/2014, 22:55:46

```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA/subTopic?skip=0&limit=100&dateMinCreation=1405544146&dateMaxCreation=1408222546
```

#### Count messages created since 8 hours

```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA?onlyCount=true&lastHourMinCreation=8
```

#### Count messages created since 3 days

* lastHourMinCreation : 24hours x 3 days = 72

```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA?onlyCount=true&lastHourMinCreation=72
```

#### Count messages created since Beginning Of Month

* dateRefCreation : Select beginning of month with "BeginningOfMonth" pattern. Start at monday.

```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA?onlyCount=true&dateRefCreation=BeginningOfMonth
```

#### Count messages created on Tuesday of current week

* dateRefCreation : Select beginning of week with "BeginningOfWeek" pattern. Start at monday.
* dateRefDeltaMinCreation : add seconds to dateRefCreation, to 60seconds x 60minutes x 24hours = 86400
* dateRefDeltaMaxCreation : add seconds to dateRefCreation, to 60seconds x 60minutes x 48hours = 172800

```bash
curl -XGET https://<tatHostname>:<tatPort>/messages/topicA?onlyCount=true&dateRefCreation=BeginningOfWeek&dateRefDeltaMinCreation=86400&dateRefDeltaMaxCreation=172800
```
