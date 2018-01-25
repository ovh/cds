---
title: "tatcli topic -h"
weight: 8
toc: true
prev: "/tatcli/tatcli-system"
next: "/tatcli/tatcli-ui"

---

## Command Description
### tatcli topic -h

```
Topic commands: tatcli topic [command]

Usage:
  tatcli topic [command]

Aliases:
  topic, t


Available Commands:
  addAdminGroup     Add Admin Groups to a topic: tatcli topic addAdminGroup [--recursive] <topic> <groupname1> [groupname2]...
  addAdminUser      Add Admin Users to a topic: tatcli topic addAdminUser [--recursive] <topic> <username1> [username2]...
  addParameter      Add Parameter to a topic: tatcli topic addParameter [--recursive] <topic> <key>:<value> [<key2>:<value2>]...
  addRoGroup        Add Read Only Groups to a topic: tatcli topic addRoGroup [--recursive] <topic> <groupname1> [<groupname2>]...
  addRoUser         Add Read Only Users to a topic: tatcli topic addRoUser [--recursive] <topic> <username1> [username2]...
  addRwGroup        Add Read Write Groups to a topic: tatcli topic addRwGroup [--recursive] <topic> <groupname1> [<groupname2>]...
  addRwUser         Add Read Write Users to a topic: tatcli topic addRwUser [--recursive] <topic> <username1> [username2]...
  allcomputelabels  Compute Labels on all topics, only for tat admin : tatcli topic allcomputelabels
  allcomputereplies Compute Replies on all topics, only for tat admin : tatcli topic allcomputereplies
  allcomputetags    Compute Tags on all topics, only for tat admin : tatcli topic allcomputetags
  allsetparam       Set a param for all topics, only for tat admin : tatcli topic allsetparam <paramName> <paramValue>
  computelabels     Compute Labels on this topic, only for tat admin and administrators on topic : tatcli topic computelabels <topic>
  computetags       Compute Tags on this topic, only for tat admin and administrators on topic : tatcli topic computetags <topic>
  create            Create a new topic: tatcli create <topic> <description of topic>
  delete            Delete a topic: tatcli delete <topic>
  deleteAdminGroup  Delete Admin Groups from a topic: tatcli topic deleteAdminGroup [--recursive] <topic> <groupname1> [<groupname2>]...
  deleteAdminUser   Delete Admin Users from a topic: tatcli topic deleteAdminUser [--recursive] <topic> <username1> [username2]...
  deleteParameter   Remove Parameter to a topic: tatcli topic deleteParameter [--recursive] <topic> <key> [<key2>]...
  deleteRoGroup     Delete Read Only Groups from a topic: tatcli topic deleteRoGroup [--recursive] <topic> <groupname1> [<groupname2>]...
  deleteRoUser      Delete Read Only Users from a topic: tatcli topic deleteRoUser [--recursive] <topic> <username1> [username2]...
  deleteRwGroup     Delete Read Write Groups from a topic: tatcli topic deleteRwGroup [--recursive] <topic> <groupname1> [<groupname2>]...
  deleteRwUser      Delete Read Write Users from a topic: tatcli topic deleteRwUser [--recursive] <topic> <username1> [username2]...
  list              List all topics: tatcli topic list [<skip>] [<limit>], tatcli topic list -h for see all criterias
  parameter         Update param on one topic: tatcli topic param [--recursive] <topic> <maxReplies> <maxLength> <canForceDate> <canUpdateMsg> <canDeleteMsg> <canUpdateAllMsg> <canDeleteAllMsg> <adminCanUpdateAllMsg> <adminCanDeleteAllMsg> <isAutoComputeTags> <isAutoComputeLabels>
  truncate          Remove all messages in a topic, only for tat admin and administrators on topic : tatcli topic truncate <topic> [--force]
  truncatelabels    Truncate Labels on this topic, only for tat admin and administrators on topic : tatcli topic truncatelabels <topic>
  truncatetags      Truncate Tags on this topic, only for tat admin and administrators on topic : tatcli topic truncatetags <topic>

Flags:
  -h, --help=false: help for topic

Global Flags: see tatcli -h

```

## Examples
### Create a Topic
```bash
tatcli topic add /topic topic description
```

### Delete a Topic
```bash
tatcli topic delete /topic
```

### Truncate a Topic
```bash
tatcli topic truncate /topic
```

### Getting Topics List
```bash
tatcli topic list
tatcli topic list skip limit
tatcli topic list skip limit true
```
if true, return nb unread messages

### Add a read only user to a topic
```bash
tatcli topic addRoUser /topic username
tatcli topic addRoUser /topic username1 username2
```
### Add a read write user to a topic
```bash
tatcli topic addRwUser /topic username
tatcli topic addRwUser /topic username1 username2
```

### Add an admin user to a topic
```bash
tatcli topic addAdminUser /topic username
tatcli topic addAdminUser /topic username1 username2
```


### Delete a read only user from a topic
```bash
tatcli topic deleteRoUser /topic username
tatcli topic deleteRoUser /topic username1 username2
```

### Delete a read write user from a topic
```bash
tatcli topic deleteRwUser /topic username
tatcli topic deleteRwUser /topic username1 username2
```

### Delete an admin user from a topic
```bash
tatcli topic deleteAdminUser /topic username
tatcli topic deleteAdminUser /topic username1 username2
```

### Add a read only group to a topic
```bash
tatcli topic addRoGroup /topic groupname
tatcli topic addRoGroup /topic groupname1 groupname2
```

### Add a read write group to a topic
```bash
tatcli topic addRwGroup /topic groupname
tatcli topic addRwGroup /topic groupname1 groupname2
```

### Add an admin group to a topic
```bash
tatcli topic addAdminGroup /topic groupname
tatcli topic addAdminGroup /topic groupname1 groupname2
```

### Delete a read only group from a topic
```bash
tatcli topic deleteRoGroup /topic groupname
tatcli topic deleteRoGroup /topic groupname1 groupname2
```

### Delete a read write group from a topic
```bash
tatcli topic deleteRwGroup /topic groupname
tatcli topic deleteRwGroup /topic groupname1 groupname2
```

### Delete an admin group from a topic
```bash
tatcli topic deleteAdminGroup /topic groupname
tatcli topic deleteAdminGroup /topic groupname1 groupname2
```
