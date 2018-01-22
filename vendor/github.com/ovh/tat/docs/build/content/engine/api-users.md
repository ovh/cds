---
title: "API - Users"
weight: 4
toc: true
prev: "/engine/api-topics"
next: "/engine/api-groups"

---

## Tat Password
It's a generated password by Tat, allowing username to communicate with Tat.
User creates an account, a mail is send to verify account and user has to go on a Tat URL to validate account and get password.
Password is encrypted in Tat Database (sha512 Sum).

First user created is an administrator.

## Create a User
Return a mail to user, with instruction to validate his account.

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -d '{"username": "userA", "fullname": "User AA", "email": "usera@foo.net", "callback": " Click on scheme://:host::port/user/verify/:username/:token to validate your account"}' \
    https://<tatHostname>:<tatPort>/user
```

Callback is a string sent by mail, indicating to the user how to validate his account.
Available fields (automatically filled by Tat ):

```
:scheme -> http of https
:host -> ip or hostname of Tat Engine
:port -> port of Tat Engine
:username -> username
:token -> tokenVerify of user
```


## Verify a User
```bash
curl -XGET \
    https://<tatHostname>:<tatPort>/user/verify/yourUsername/tokenVerifyReceivedByMail
```
This url can be called only once per password and expired 30 minutes after querying create user with POST on `/user`

## Ask for reset a password
Returns: tokenVerify by email

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -d '{"username": "userA", "email": "usera@foo.net"}' \
    https://<tatHostname>:<tatPort>/user/reset
```

## Get information about current User
```bash
curl -XGET \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me
```


## Get contacts

Retrieves contacts presences since n seconds

Example since 15 seconds :

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/contacts/15
```

## Add a contact
```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/contact/username
```

## Remove a contact
```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/contacts/username
```


## Add a favorite topic
```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/topics/myTopic/sub-topic
```

## Remove a favorite topic
```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/topics/myTopic/sub-topic
```

## Enable notifications on one topic
```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/enable/notifications/topics/myTopic/sub-topic
```

## Disable notifications on one topic
```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/disable/notifications/topics/myTopic/sub-topic
```

## Enable notifications on all topics

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/enable/notifications/alltopics
```

## Disable notifications on all topics, except /Private/*

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/disable/notifications/alltopics
```

## Add a favorite tag

```bash
curl -XPOST \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/tags/myTag
```

## Remove a favorite tag
```bash
curl -XDELETE \
    -H "Content-Type: application/json" \
    -H "Tat_username: userA" \
    -H "Tat_password: password" \
    https://<tatHostname>:<tatPort>/user/me/tags/myTag
```

## Getting Users List
```bash
curl -XGET https://<tatHostname>:<tatPort>/users?skip=<skip>&limit=<limit> | python -m json.tool
curl -XGET https://<tatHostname>:<tatPort>/users?skip=<skip>&limit=<limit>&argName=valName&arg2Name=val2Name | python -m json.tool
```

Users list with groups (admin only)
```bash
curl -XGET https://<tatHostname>:<tatPort>/users?skip=<skip>&limit=<limit>&withGroups=true
```

### Parameters

* skip: Skip skips over the n initial documents from the query results
* limit: Limit restricts the maximum number of documents retrieved
* username: Username
* fullname: Fullname
* dateMinCreation: filter result on dateCreation, timestamp Unix format
* dateMaxCreation: filter result on dateCreation, timestamp Unix Format


### Example
```bash
curl -XGET https://<tatHostname>:<tatPort>/users?skip=0&limit=100 | python -m json.tool
```

## Convert a user to a system user
Only for Tat Admin: convert a `normal user` to a `system user`.
A system user must have a username starting with `tat.system`.
Remove email and set user attribute IsSystem to true.
This action returns a new password for this user.
Warning: it is an irreversible action.

Flag `canWriteNotifications` allows (or not if false) the `system user` to write inside private topics of user `/Private/username/Notifications`

Flag `canListUsersAsAdmin` allows this `system user` to view all user's fields (email, etc...)

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToConvert", "canWriteNotifications": "true", "canListUsersAsAdmin": "true" }' \
    https://<tatHostname>:<tatPort>/user/convert
```

## Update flags on system user
Only for Tat Admin.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "userSystem", "canWriteNotifications": "true", "canListUsersAsAdmin": "true" }' \
    https://<tatHostname>:<tatPort>/user/updatesystem
```

## Reset a password for system user
Only for Tat Admin.
A `system user` must have a username starting with `tat.system`.
This action returns a new password for this user.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "userameSystemToReset" }' \
    https://<tatHostname>:<tatPort>/user/resetsystem
```


## Grant a user to an admin user
Only for Tat Admin: convert a `normal user` to an `admin user`.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToGrant" }' \
    https://<tatHostname>:<tatPort>/user/setadmin
```

## Rename a username
Only for Tat Admin: rename the username of a user. This action updates all Private topics of the user.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToRename", "newUsername": "NewUsername" }' \
    https://<tatHostname>:<tatPort>/user/rename
```

## Update fullname or email
Only for Tat Admin: update fullname and email of a user.

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToRename", "newFullname": "NewFullname", "newEmail": "NewEmail" }' \
    https://<tatHostname>:<tatPort>/user/update
```

## Archive a user
Only for Tat Admin

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToRename" }' \
    https://<tatHostname>:<tatPort>/user/archive
```

## Check Private Topics and Default Group on one user
Only for Tat Admin

```bash
curl -XPUT \
    -H "Content-Type: application/json" \
    -H "Tat_username: userAdmin" \
    -H "Tat_password: passwordAdmin" \
    -d '{ "username": "usernameToRename",  "fixPrivateTopics": true, "fixDefaultGroup": true }' \
    https://<tatHostname>:<tatPort>/user/check
```
