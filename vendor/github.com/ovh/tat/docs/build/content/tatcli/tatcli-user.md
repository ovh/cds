---
title: "tatcli users -h"
weight: 11
toc: true
prev: "/tatcli/tatcli-update"
next: "/tatcli/tatcli-version"

---

## Command Description
### tatcli user -h

```
User commands: tatcli user <command>

Usage:
  tatcli user [command]

Aliases:
  user, u


Available Commands:
  list                          List all users: tatcli user list [<skip>] [<limit>]
  me                            Get Information about you: tatcli user me
  contacts                      Get contacts presences since n seconds: tatcli user contacts <seconds>
  addContact                    Add a contact: tatcli user addContact <contactUsername>
  removeContact                 Remove a contact: tatcli user removeContact <contactUsername>
  addFavoriteTopic              Add a favorite Topic: tatcli user addFavoriteTopic <topicName>
  removeFavoriteTopic           Remove a favorite Topic: tatcli user removeFavoriteTopic <topicName>
  enableNotificationsTopic      Enable notifications on a topic: tatcli user enableNotificationsTopic <topicName>
  enableNotificationsAllTopics  Enable notifications on a topic: tatcli user enableNotificationsAllTopics
  disableNotificationsTopic     Disable notifications on a topic: tatcli user disableNotificationsTopic <topicName>
  disableNotificationsAllTopics Disable notifications on all topics: tatcli user disableNotificationsAllTopics
  addFavoriteTag                Add a favorite Tag: tatcli user addFavoriteTag <tag>
  removeFavoriteTag             Remove a favorite Tag: tatcli user removeFavoriteTag <tag>
  add                           Add a user: tatcli user add <username> <email> <fullname>
  reset                         Ask for Reset a password: tatcli user reset <username> <email>
  resetSystemUser               Reset password for a system user (admin only): tatcli user resetSystemUser <username>
  convert                       Convert a user to a system user (admin only): tatcli user convert <username> <canWriteNotifications> <canListUsersAsAdmin>
  updateSystemUser              Update a system user (admin only): tatcli user updateSystemUser <username> <canWriteNotifications> <canListUsersAsAdmin>
  archive                       Archive a user (admin only): tatcli user archive <username>
  rename                        Rename username of a user (admin only): tatcli user rename <oldUsername> <newUsername>
  update                        Update Fullname and Email of a user (admin only): tatcli user update <username> <newEmail> <newFullname>
  setAdmin                      Grant user to Tat admin (admin only): tatcli user setAdmin <username>
  verify                        Verify account: tatcli user verify [--save] <username> <tokenVerify>
  check                         Check Private Topics and Default Group on one user (admin only): tatcli user check <username> <fixPrivateTopics> <fixDefaultGroup>

Flags:
  -h, --help=false: help for user

Global Flags: see tatcli -h

```

## Examples
### Create a user
```bash
tatcli user add username email fullname
```

### Verify account
```bash
tatcli user verify username tokenVerify
```

For saving configuration in $HOME/.tatcli/config.json file
```bash
tatcli user verify --save username tokenVerify
```

### Ask for reset password
```bash
tatcli user reset username email
```

### Get information about me
```bash
tatcli user me
```

### Get contacts presences since n seconds: tatcli user contacts <seconds>
```bash
tatcli user contacts 15
```

### Add a favorite tag
```bash
tatcli user addFavoriteTag myTag
```

### Remove a favorite tag
```bash
tatcli user removeFavoriteTag myTag
```

### Add a favorite topic
```bash
tatcli user addFavoriteTopic /topic/sub-topic
```

### Remove a favorite topic
```bash
tatcli user removeFavoriteTopic /topic/sub-topic
```

### Enable notifications on a topic

Notifications are by default enabled on topic

```bash
tatcli user enableNotificationsTopic /topic/sub-topic
```

### Disable notifications on a topic
```bash
tatcli user disableNotificationsTopic /topic/sub-topic
```

### Enable notifications on all topics

Notifications are by default enabled on all topics

```bash
tatcli user enableNotificationsAllTopics
```

### Disable notifications on all topics
```bash
tatcli user disableNotificationsAllTopics
```


### List Users
```bash
tatcli user list
```

with groups (admin only):

```bash
tatcli user list --withGroups
```

### Convert to a system user (Admin only)
```bash
tatcli user convert usernameToConvertSystem flagCanWriteOnNotificationsTopics flagCanListUsersAsAdmin
```
flagCanWriteOnNotificationsTopics could be true or false

flagCanListUsersAsAdmin could be true or false


### Update a system user (Admin only)
```bash
tatcli user userSystemUser usernameOfSystemUser flagCanWriteOnNotificationsTopics flagCanListUsersAsAdmin
```

flagCanWriteOnNotificationsTopics could be true or false

flagCanListUsersAsAdmin could be true or false

### Grant a user to Tat Admin (Admin only)
```bash
tatcli user setAdmin usernameToGrant
```

### Archive a user (Admin only)
```bash
tatcli user archive usernameToArchive
```

### Rename a username  (Admin only)
```bash
tatcli user rename oldUsername newUsername
```

### Update fullname and email (Admin only)
```bash
tatcli user update username newEmail newFirstname newLastname
```

### Check a user (Admin only)

Check Private Topics and Default Group on one user:

```bash
tatcli user check <username> <fixPrivateTopics> <fixDefaultGroup>
```

Example :

```bash
tatcli check username true true
```
