---
title: "API - Topics"
weight: 3
toc: true
prev: "/engine/api-messages"
next: "/engine/api-users"

---

## Create a Topic

Rules:

* User can create a root topic if he is a Tat Admin.
* User can create topics under `/Private/username/`
* User can create topics if he is an admin on the Parent Topic or belong to an admin group on the Parent topic.
Example:  Create /AAA/BBB: Parent Topic is /AAA

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "description": "Topic Description"}' \
    https://<tatHostname>:<tatPort>/topic
```

## Delete a topic
```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/topic/subtopic
```

## Truncate a topic

Only for Tat Admin and administrators on topic.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA"}' \
    https://<tatHostname>:<tatPort>/topic/truncate
```

## Compute tags on a topic

Only for Tat Admin and administrators on topic.

Set "tags" attribute on topic, with an array of all tags used in this topic.
One entry in "tags" attribute per text of tag.

Topic's tags are showed with :
GET https://<tatHostname>:<tatPort>/topic/topicName

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA"}' \
    https://<tatHostname>:<tatPort>/topic/compute/tags
```

Example of usage of tags attribute: autocompletion of tag on UI when written new message on a topic

## Compute labels on a topic

Only for Tat Admin and administrators on topic.

Set "labels" attribute on topic, with an array of all labels used in this topic.
One entry in "labels" attribute per text & color of label.

Topic's labels are showed with :
GET https://<tatHostname>:<tatPort>/topic/topicName

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA"}' \
    https://<tatHostname>:<tatPort>/topic/compute/labels
```

Example of usage of labels attribute: label autocompletion on UI when adding new label

## Compute tags on all topics

Only for Tat Admin.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/topics/compute/tags
```

## Compute labels on all topics

Only for Tat Admin.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/topics/compute/labels
```

## Set a param on all topics

Only for Tat Admin and for attributes isAutoComputeTags and isAutoComputeLabels.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"paramName":"isAutoComputeLabels","paramValue":"false"}' \
    https://<tatHostname>:<tatPort>/topics/param
```


## Truncate cached tags on a topic

Only for Tat Admin and administrators on topic.

Truncate "tags" attribute on topic.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA"}' \
    https://<tatHostname>:<tatPort>/topic/tags/truncate
```

## Truncate cached labels on a topic

Only for Tat Admin and administrators on topic.

Truncate "labels" attribute on topic.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA"}' \
    https://<tatHostname>:<tatPort>/topic/labels/truncate
```

## Getting one Topic
```bash
curl -XGET https://<tatHostname>:<tatPort>/topic/topicName | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/topic/topicName/subTopic | python -m json.tool
```

## Getting Topics List
```bash
curl -XGET https://<tatHostname>:<tatPort>/topics?skip=<skip>&limit=<limit> | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/topics?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name | python -m json.tool
```

### Parameters
* skip: Skip skips over the n initial documents from the query results
* limit: Limit restricts the maximum number of documents retrieved
* topic: Topic name, example: /topicA
* topicPath: Topic start path, example: /topicA will return /topicA/subA, /topicA/subB
* idTopic: id of topic
* description: description of topic
* dateMinCreation: filter result on dateCreation, timestamp Unix format
* dateMaxCreation: filter result on dateCreation, timestamp Unix Format
* getNbMsgUnread: if true, add new array to return, topicsMsgUnread with topic:flag. flag can be -1 if unknown, 0 or 1 if there is one or more messages unread
* onlyFavorites: if true, return only favorites topics, except /Private/*. All privates topics are returned.
* getForTatAdmin: if true, and requester is a Tat Admin, returns all topics (except /Private/*) without checking user access


### Example
```bash
curl -XGET https://<tatHostname>:<tatPort>/topics?skip=0&limit=100 | python -m json.tool
```

## Add a parameter to a topic

For admin of topic or on `/Private/username/*`

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "key": "keyOfParameter", "value": "valueOfParameter", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/parameter
```

## Remove a parameter to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "key": "keyOfParameter", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/parameter
```

## Add a read only user to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/rouser
```

## Add a read write user to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/rwuser
```

## Add an admin user to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/adminuser
```

## Delete a read only user from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/rouser
```

## Delete a read write user from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/wuser
```

## Delete an admin user from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "username": "usernameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/adminuser
```

## Add a read only group to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/rogroup
```

## Add a read write group to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/rwgroup
```

## Add an admin group to a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToAdd", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/add/admingroup
```


## Delete a read only group from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/rogroup
```

## Delete a read write group from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/rwgroup
```

## Delete an admin group from a topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "groupname": "groupnameToRemove", "recursive": "false"}' \
    https://<tatHostname>:<tatPort>/topic/remove/rwgroup
```


## Update param on one topic: admin or admin on topic
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic": "/topicA", "recursive": "false", "maxlength": 140, "maxreplies": 30, "canForceDate": false, "canUpdateMsg": false, "canDeleteMsg": false, "canUpdateAllMsg": false, "canDeleteAllMsg": false, "adminCanUpdateAllMsg": false, "adminCanDeleteAllMsg": false}' \
    https://<tatHostname>:<tatPort>/topic/param
```

Parameters key is optional.

Example with key parameters :

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"topic":"/Internal/Alerts","recursive":false,"maxlength":300,"maxreplies":30,"canForceDate":false,"canUpdateMsg":false,"canDeleteMsg":true,"canUpdateAllMsg":false,"canDeleteAllMsg":false,"adminCanUpdateAllMsg":false,"adminCanDeleteAllMsg":false,"parameters":[{"key":"agileview","value":"qsdf#qsdf"},{"key":"tatwebui.view.default","value":"standardview-list"},{"key":"tatwebui.view.forced","value":""}]}' \
    https://<tatHostname>:<tatPort>/topic/param
```
