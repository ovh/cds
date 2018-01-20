---
title: "API - Groups"
weight: 5
toc: true
prev: "/engine/api-users"
next: "/engine/api-presences"

---

## Create a group

Only for Tat Admin

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"name": "groupName", "description": "Group Description"}' \
    https://<tatHostname>:<tatPort>/group
```

## Update a group

Only for Tat Admin

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"newName": "groupName", "newDescription": "Group Description"}' \
    https://<tatHostname>:<tatPort>/group/<groupName>
```

## Getting groups List

```bash
curl -XGET https://<tatHostname>:<tatPort>/groups?skip=<skip>&limit=<limit> | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/groups?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name | python -m json.tool
```

### Parameters

* skip: Skip skips over the n initial documents from the query results
* limit: Limit restricts the maximum number of documents retrieved
* idGroup: Id Group
* name: Name of group
* description: Description of group
* dateMinCreation: filter result on dateCreation, timestamp Unix format
* dateMaxCreation: filter result on dateCreation, timestamp Unix Format

## Delete a group

Only for Tat Admin

```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    https://<tatHostname>:<tatPort>/group/<groupName>
```

## Add a user to a group
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"groupname": "groupName", "username": "usernameToAdd"}' \
    https://<tatHostname>:<tatPort>/group/add/user
```

## Delete a user from a group
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"groupname": "groupName", "username": "usernameToAdd"}' \
    https://<tatHostname>:<tatPort>/group/remove/user
```


## Add an admin user to a group
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"groupname": "groupName", "username": "usernameToAdd"}' \
    https://<tatHostname>:<tatPort>/group/add/adminuser
```

## Delete an admin user from a group
```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: admin" \
    -H "Tat_password: passwordAdmin" \
    -d '{"groupname": "groupName", "username": "usernameToAdd"}' \
    https://<tatHostname>:<tatPort>/group/remove/adminuser
```
