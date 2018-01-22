---
title: "tatcli message -h"
weight: 4
toc: true
prev: "/tatcli/tatcli-group"
next: "/tatcli/tatcli-presence"

---

## Command Description
### tatcli message -h

```
Manipulate messages: tatcli message <command>

Usage:
  tatcli message [command]

Aliases:
  message, m, msg


Available Commands:
  add         tatcli message add [--dateCreation=timestamp] <topic> <my message>
  concat      Update a message (if it's enabled on topic) by adding additional text at the end of message: tatcli message concat <topic> <idMessage> <additional text...>
  delete      Delete a message: tatcli message delete <topic> <idMessage> [--cascade] [--cascadeForce]
  deletebulk  Delete a list of messages: tatcli message deletebulk <topic> <skip> <limit> [--cascade] [--cascadeForce]
  label       Add a label to a message: tatcli message label <topic> <idMessage> <colorInHexa> <my Label>
  like        Like a message: tatcli message like <topic> <idMessage>
  list        List all messages on one topic: tatcli msg list <Topic> <skip> <limit>
  move        Move a message: tatcli message move <oldTopic> <idMessage> <newTopic>
  relabel     Remove all labels and add new ones to a message: tatcli msg relabel <topic> <idMessage> --label="#EEEE;myLabel1,#EEEE;myLabel2" --options="myLabelToRemove1,myLabelToRemove2"
  reply       Reply to a message: tatcli message reply <topic> <inReplyOfId> <my message...>
  task        Create a task from one message: tatcli message task /Private/username/tasks/sub-topic idMessage
  unlabel     Remove a label from a message: tatcli message unlabel <topic> <idMessage> <my Label>
  unlike      Unlike a message: tatcli message unlike <topic> <idMessage>
  untask      Remove a message from tasks: tatcli message untask /Private/username/tasks idMessage
  unvotedown  Remove a vote down from a message: tatcli message unvotedown <topic> <idMessage>
  unvoteup    Remove a vote UP from a message: tatcli message unvoteup <topic> <idMessage>
  update      Update a message (if it's enabled on topic): tatcli message update <topic> <idMessage> <my message...>
  votedown    Vote Down a message: tatcli message votedown <topic> <idMessage>
  voteup      Vote UP a message: tatcli message voteup <topic> <idMessage>


```

### Command Message list

```
List all messages on one topic: tatcli msg list <Topic> <skip> <limit>

Usage:
  tatcli message list [flags]

Aliases:
  list, l


Flags:
      --allIDMessage string              Search in All ID Message (idMessage, idReply, idRoot)
      --andLabel string                  Search by label (and) : could be labelA,labelB
      --andTag string                    Search by tag (and) : could be tagA,tagB
      --dateMaxCreation string           Search by dateCreation (timestamp), select messages where dateCreation <= dateMaxCreation
      --dateMaxUpdate string             Search by dateUpdate (timestamp), select messages where dateUpdate <= dateMaxUpdate
      --dateMinCreation string           Search by dateCreation (timestamp), select messages where dateCreation >= dateMinCreation
      --dateMinUpdate string             Search by dateUpdate (timestamp), select messages where dateUpdate >= dateMinUpdate
      --dateRefCreation string           This have to be used with dateRefDeltaMinCreation and / or dateRefDeltaMaxCreation. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear
      --dateRefDeltaMaxCreation string   Add seconds to dateRefCreation flag
      --dateRefDeltaMaxUpdate string     Add seconds to dateRefUpdate flag
      --dateRefDeltaMinCreation string   Add seconds to dateRefCreation flag
      --dateRefDeltaMinUpdate string     Add seconds to dateRefUpdate flag
      --dateRefUpdate string             This have to be used with dateRefDeltaMinUpdate and / or dateRefDeltaMaxUpdate. This could be BeginningOfMinute, BeginningOfHour, BeginningOfDay, BeginningOfWeek, BeginningOfMonth, BeginningOfQuarter, BeginningOfYear
      --exec stringSlice                 --stream required. Exec a cmd on each new message: --stream --exec 'myLights --pulse blue --duration=1000' With only --onlyMsgCount=true : --exec min:max:cmda --exec min:max:cmdb, example: --exec 0:4:'cmdA' --exec 5::'cmdb'
      --execErr stringSlice              --stream required. Exec a cmd on each error while requesting tat: --stream --exec 'myLights --pulse blue --duration=1000' --execErr 'myLights --pulse red --duration=2000'
      --idMessage string                 Search by IDMessage
      --inReplyOfID string               Search by IDMessage InReply
      --inReplyOfIDRoot string           Search by IDMessage IdRoot
      --label string                     Search by label: could be labelA,labelB
      --lastHourMaxCreation string       Search by dateCreation, select messages where dateCreation <= Now Beginning Of Hour - (60 * lastHourMaxCreation)
      --lastHourMaxUpdate string         Search by dateUpdate, select messages where dateUpdate <= Now Beginning Of Hour - (60 * lastHourMaxCreation)
      --lastHourMinCreation string       Search by dateCreation, select messages where dateCreation >= Now Beginning Of Hour - (60 * lastHourMinCreation)
      --lastHourMinUpdate string         Search by dateUpdate, select messages where dateUpdate >= Now Beginning Of Hour - (60 * lastHourMinCreation)
      --lastMaxCreation string           Search by dateCreation (duration in second), select messages where dateCreation <= now - lastMaxCreation
      --lastMaxUpdate string             Search by dateUpdate (duration in second), select messages where dateUpdate <= now - lastMaxCreation
      --lastMinCreation string           Search by dateCreation (duration in second), select messages where dateCreation >= now - lastMinCreation
      --lastMinUpdate string             Search by dateUpdate (duration in second), select messages where dateUpdate >= now - lastMinCreation
      --limitMaxNbReplies string         In onetree mode, filter root messages with min or equals maxNbReplies
      --limitMaxNbVotesDown string       Search by nbVotesDown
      --limitMaxNbVotesUP string         Search by nbVotesUP
      --limitMinNbReplies string         In onetree mode, filter root messages with more or equals minNbReplies
      --limitMinNbVotesDown string       Search by nbVotesDown
      --limitMinNbVotesUP string         Search by nbVotesUP
      --notLabel string                  Search by label (exclude): could be labelA,labelB
      --notTag string                    Search by tag (exclude) : could be tagA,tagB
      --onlyCount string                 --onlyCount=true: only count messages, without retrieve msg. limit, skip, treeview criterias are ignored.
      --onlyMsgReply string              --onlyMsgReply=true: restricts to reply message only (inReplyOfIDRoot not empty). If treeView is used, limit search criteria to reply, messages root are still given, independently of search criteria.
      --onlyMsgRoot string               --onlyMsgRoot=true: restricts to root message only (inReplyOfIDRoot empty). If treeView is used, limit search criteria to root message, replies are still given, independently of search criteria.
      --sortBy string                    --sortBy=-dateCreation: sort message. Use '-' to reverse sort. Default is --sortBy=-dateCreation. You can use: text, topic, inReplyOfID, inReplyOfIDRoot, nbLikes, labels, likers, votersUP, votersDown, nbVotesUP, nbVotesDown, userMentions, urls, tags, dateCreation, dateUpdate, author, nbReplies
      --startLabel string                Search by a label prefix: --startLabel='mykey:,myKey2:'
      --startTag string                  Search by a tag prefix: --startTag='mykey:,myKey2:'
  -s, --stream                           stream messages --stream. Request tat each 10s, default sort: dateUpdate
      --streamFormat string              --stream required. Format output. Available:  $TAT_MSG_ID $TAT_MSG_TEXT $TAT_MSG_TOPIC $TAT_MSG_INREPLYOFID $TAT_MSG_INREPLYOFIDROOT $TAT_MSG_NBLIKES $TAT_MSG_NBVOTESUP $TAT_MSG_NBVOTESDOWN $TAT_MSG_DATECREATION $TAT_MSG_DATECREATION_HUMAN $TAT_MSG_DATEUPDATE $TAT_MSG_DATEUPDATE_HUMAN $TAT_MSG_AUTHOR_USERNAME $TAT_MSG_NBREPLIES $TAT_MSG_LABELS $TAT_MSG_TAGS $TAT_MSG_LIKERS $TAT_MSG_VOTERSUP $TAT_MSG_VOTERSDOWN $TAT_MSG_USERMENTIONS $TAT_MSG_URLS (default "$TAT_MSG_DATEUPDATE_HUMAN $TAT_MSG_AUTHOR_USERNAME $TAT_MSG_TEXT")
      --tag string                       Search by tag : could be tagA,tagB
      --text string                      Search by text
      --topic string                     Search by topic
      --treeView string                  Tree View of messages: onetree or fulltree. Default: notree
      --username string                  Search by username : could be usernameA,usernameB

```

## Examples

### Create a message
```bash
tatcli message add /YourTopic my message
```

With labels:

```bash
tatcli msg add --label="#cccccc;label,#dddddd;label2" /YourTopic my message
```

If you are a `system user`, you can force date creation. Date as timestamp

```bash
tatcli message add --dateCreation=11111 /YourTopic my message
```

### Reply to a message
```bash
tatcli message reply /YourTopic idOfMessage my message
```

### Like a message
```bash
tatcli message like /YourTopic idOfMessage
```

### Unlike a message
```bash
tatcli message unlike /YourTopic idOfMessage
```

### Add a label to a message
```bash
tatcli message label /YourTopic idOfMessage color myLabel
```

### Remove a label from a message
```bash
tatcli message unlabel /YourTopic idOfMessage myLabel
```

### Remove all labels and add new ones to a message
```bash
tatcli message relabel /YourTopic idOfMessage --label="#cccccc;label,#dddddd;label2"
```

### Update a message by adding additional text at the end of message
```bash
tatcli message concat /YourTopic idOfMessage additional text
```

### Vote UP a message
```bash
tatcli message voteup /YourTopic idOfMessage
```

### Vote Down a message
```bash
tatcli message votedown /YourTopic idOfMessage
```

### Remove a Vote UP from a message
```bash
tatcli message unvoteup /YourTopic idOfMessage
```

### Remove a Vote Down from a message
```bash
tatcli message unvotedown /YourTopic idOfMessage
```

### Create a task from one message
```bash
tatcli message task /Private/username/Tasks idOfMessage
```

### Remove a message from tasks
```bash
tatcli message untask /Private/username/Tasks idOfMessage
```

### Move a message to another topic
```bash
tatcli message move /MyOldTopic/SubTopic idOfMessage /MyNewTopic/SubTopic
```

### Getting message
```bash
tatcli message list /YourTopic
tatcli message list /YourTopic 0 10
```

### Stream messages and exec a command on each message

Execute 'myLights' binary for each new message on topic /YourTopic/YourSubTopic

```bash
tatcli msg list /YourTopic/YourSubTopic --stream --exec "./myLights --pulse blue --duration=1000"
```

Execute 'myLights' binary for each new message on topic /YourTopic/YourSubTopic and
another command if there is an error while requesting tat.

```bash
tatcli msg list /YourTopic/YourSubTopic --stream --exec "./myLights --pulse blue --duration=1000" --exec "./myLights --pulse red --duration=2000"
```

### Stream messages and exec a command on count value

Count messages with label open on topic /Internal/Alerts, then :

* if there is 0 message, execute `blinkstick --morph green`
* if there are between 1 and 5 messages, execute `blinkstick --morph yellow`
* if there are more than 5 messages, execute `blinkstick --morph red`
* if there is an error while requesting tat, execute `blinkstick --pulse red --duration=2000`

```bash
tatcli msg list /Internal/Alerts --label=open --stream --exec 0:0:'blinkstick --morph green' --exec 1:5:'blinkstick --morph blue' --exec 6::'blinkstick --morph yellow' --execErr "blinkstick --pulse red --duration=2000"
```

### Stream messages, format output and exec with arguments

Stream messages and display only username and text.
```bash
tatcli msg list /Internal/Alerts --stream --streamFormat '$TAT_MSG_AUTHOR_USERNAME $TAT_MSG_TEXT '
```

Stream messages and execute foo.sh on each message with 2 arguments: message ID and message Text.

```bash
echo 'echo $* >> foo.log' > foo.sh && chmod +x foo.sh

tatcli msg list /Internal/Alerts --stream --exec './foo.sh $TAT_MSG_ID $TAT_MSG_TEXT'
```

You can use :

* `$TAT_MSG_ID`
* `$TAT_MSG_TEXT`
* `$TAT_MSG_TOPIC`
* `$TAT_MSG_INREPLYOFID`
* `$TAT_MSG_INREPLYOFIDROOT`
* `$TAT_MSG_NBLIKES`
* `$TAT_MSG_NBVOTESUP`
* `$TAT_MSG_NBVOTESDOWN`
* `$TAT_MSG_DATECREATION`
* `$TAT_MSG_DATECREATION_HUMAN`
* `$TAT_MSG_DATEUPDATE`
* `$TAT_MSG_DATEUPDATE_HUMAN`
* `$TAT_MSG_AUTHOR_USERNAME`
* `$TAT_MSG_NBREPLIES`
* `$TAT_MSG_LABELS`
* `$TAT_MSG_TAGS`
* `$TAT_MSG_LIKERS`
* `$TAT_MSG_VOTERSUP`
* `$TAT_MSG_VOTERSDOWN`
* `$TAT_MSG_USERMENTIONS`
* `$TAT_MSG_URLS`


#### Example : Green if new UP alert, Orange if new AL alert on message with FOO tag

tatcli msg list /Internal/Alerts –tag FOO --stream --exec './blink.sh $TAT_MSG_TAG'

with blink.sh :

```
#!/bin/bash

for tag in "$*"
do
test "${tag#*AL}" != "$tag" && blinkstick --pulse red --duration=1000 && exit 0
test "${tag#*UP}" != "$tag" && blinkstick --pulse blue --duration=1000 && exit 0
done

exit 1;
```
