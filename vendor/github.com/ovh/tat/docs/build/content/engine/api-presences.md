---
title: "API - Presences"
weight: 6
toc: true
prev: "/engine/api-groups"
next: "/engine/api-system"

---

## Add presence
Status could be: `online`, `offline`, `busy`.

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	-d '{ "status": "online" }' \
	https://<tatHostname>:<tatPort>/presenceget/topic/sub-topic
```

## Getting Presences
```bash
curl -XGET https://<tatHostname>:<tatPort>/presences/<topic>?skip=<skip>&limit=<limit> | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/presences/<topic>?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name | python -m json.tool
```

## Parameters

* `topic:` /yourTopic/subTopic
* `skip`: Skip skips over the n initial presences from the query results
* `limit`: Limit restricts the maximum number of presences retrieved
* `status`: status: `online`, `offline`, `busy`
* `dateMinPresence`: filter result on datePresence, timestamp Unix format
* `dateMaxPresence`: filter result on datePresence, timestamp Unix Format
* `username`: username to search


### Examples
```bash
curl -XGET https://<tatHostname>:<tatPort>/presences/topicA?skip=0&limit=100 | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/presences/topicA/subTopic?skip=0&limit=100&dateMinPresence=1405544146&dateMaxPresence=1405544146 | python -m json.tool
```

## Delete presence
Admin can delete presences a another user on one topic.
Users can delete their own presence.

```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: username" \
    -H "Tat_password: passwordOfUser" \
	https://<tatHostname>:<tatPort>/presences/topic/sub-topic
```
