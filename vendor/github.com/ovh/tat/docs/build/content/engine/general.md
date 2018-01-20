---
title: "General"
weight: 1
toc: true
prev: "/engine"
next: "/engine/api-messages"

---


## General Specifications

* Topic
 * Contains 0 or n messages
 * Administrator(s) of Topic can create Topic inside it
* Message
 * Consists of text, tags and labels
 * Can not be deleted or modified (by default)
 * Is limited in characters (topic setting)
 * Is always attached to one topic
* Tag
 * Within the message content
 * Can not be added after message creation (by default)
* Label
 * Can be added or removed freely
 * Have a color
* Group
 * Managed by an administrator(s): adding or removing users from the group
 * Without prior authorization, a group or user has no access to topics
 * A group or a user can be read-only or read-write on a topic
* Task
 * A *task* is a message that is both in the topic task of a user and in the original topic
* Administrator(s)
 * Tat Administrator: all configuration access
 * On Group(s): can add/remove member(s)
 * On Topic(s): can create Topic inside it, update parameters

## Some rules and rules exception
* Deleting a message is possible in the private topics, or can be granted on other topic
* Modification of a message is possible in private topics, or can be granted on other topic
* The default length of a message is 140 characters, this limit can be modified by topic
* A date creation of a message can be explicitly set by a system user
* message.dateCreation and message.dateUpdate are in timestamp format, ex:
 * 1436912447: 1436912447 seconds
 * 1436912447.345678: 1436912447 seconds and 345678 milliseconds

## Detailed Specifications
* Topic
 * addAdminGroup     [Engine](/engine/api-topics/#add-an-admin-group-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-an-admin-group-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddAdminGroups)
 * addAdminUser      [Engine](/engine/api-topics/#add-an-admin-user-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-an-admin-user-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddAdminUsers)     
 * addParameter      [Engine](/engine/api-topics/#add-a-parameter-to-a-topic) [Tatcli](/tatcli/tatcli-topic/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddParameter)     
 * addRoGroup        [Engine](/engine/api-topics/#add-a-read-only-group-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-a-read-only-group-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddRoGroups)
 * addRoUser         [Engine](/engine/api-topics/#add-a-read-only-user-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-a-read-only-user-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddRoUsers)
 * addRwGroup        [Engine](/engine/api-topics/#add-a-read-write-group-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-a-read-write-group-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddRwGroups)
 * addRwUser         [Engine](/engine/api-topics/#add-a-read-write-user-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-a-read-write-user-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAddRwUsers)
 * allcomputelabels  [Engine](/engine/api-topics/#compute-labels-on-all-topics) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAllComputeLabels)
 * allcomputereplies [Engine]() [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAllComputeReplies)
 * allcomputetags    [Engine](/engine/api-topics/#compute-tags-on-all-topics) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAllComputeTags)
 * allsetparam       [Engine](/engine/api-topics/#set-a-param-on-all-topics) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicAllSetParam)
 * computelabels     [Engine](/engine/api-topics/#compute-labels-on-a-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicComputeLabels)
 * computetags       [Engine](/engine/api-topics/#compute-tags-on-a-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicComputeTags)
 * create            [Engine](/engine/api-topics/#create-a-topic) [Tatcli](/tatcli/tatcli-topic/#create-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicCreate)
 * delete            [Engine](/engine/api-topics/#delete-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDelete)
 * deleteAdminGroup  [Engine](/engine/api-topics/#delete-an-admin-group-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-an-admin-group-from-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteAdminGroups)
 * deleteAdminUser   [Engine](/engine/api-topics/#delete-an-admin-user-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-an-admin-user-from-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteAdminUsers)
 * deleteParameter   [Engine](/engine/api-topics/#remove-a-parameter-to-a-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteParameters)
 * deleteRoGroup     [Engine](/engine/api-topics/#delete-a-read-only-group-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-a-read-only-group-from-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteRoGroups)
 * deleteRoUser      [Engine](/engine/api-topics/#delete-a-read-only-user-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#add-a-read-only-user-to-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteRoUsers)
 * deleteRwGroup     [Engine](/engine/api-topics/#delete-a-read-write-group-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-a-read-write-group-from-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteRwGroups)
 * deleteRwUser      [Engine](/engine/api-topics/#delete-a-read-write-user-from-a-topic) [Tatcli](/tatcli/tatcli-topic/#delete-a-read-write-user-from-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicDeleteRwUsers)
 * list              [Engine](/engine/api-topics/#getting-topics-list) [Tatcli](/tatcli/tatcli-topic/#getting-topics-list) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicList)
 * oneTopic          [Engine](/engine/api-topics/#getting-one-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicOne)
 * parameter         [Engine](/engine/api-topics/#update-param-on-one-topic-admin-or-admin-on-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicParameter)
 * removeFilter      [Engine]() [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicRemoveFilter)
 * truncate          [Engine](/engine/api-topics/#truncate-a-topic) [Tatcli](/tatcli/tatcli-topic/#truncate-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicTruncate)
 * truncatelabels    [Engine](/engine/api-topics/#truncate-cached-labels-on-a-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicTruncateLabels)
 * truncatetags      [Engine](/engine/api-topics/#truncate-cached-tags-on-a-topic) [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicTruncateTags)
 * updateFilter      [Engine]() [Tatcli](/tatcli/tatcli-topic/#tatcli-topic-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.TopicUpdateFilter)
* Message
 * add         [Engine](/engine/api-messages/#store-a-new-message) [Tatcli](/tatcli/tatcli-message/#create-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageAdd)
 * addbulk     [Engine](/engine/api-messages/#store-some-messages) [Tatcli](/tatcli/tatcli-message/#tatcli-message-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageAddBulk)
 * concat      [Engine](/engine/api-messages/#concat-a-message-adding-additional-text-to-one-message) [Tatcli](/tatcli/tatcli-message/#update-a-message-by-adding-additional-text-at-the-end-of-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageConcat)
 * count       [Engine](/engine/api-messages/#parameters) [Tatcli](/tatcli/tatcli-message/#tatcli-message-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageCount)
 * delete      [Engine](/engine/api-messages/#delete-a-message) [Tatcli](/tatcli/tatcli-message/#tatcli-message-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageDelete)
 * deletebulk  [Engine](/engine/api-messages/#delete-a-list-of-messages) [Tatcli](/tatcli/tatcli-message/#tatcli-message-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessagesDeleteBulk)
 * label       [Engine](/engine/api-messages/#add-a-label-to-a-message) [Tatcli](/tatcli/tatcli-message/#add-a-label-to-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageLabel)
 * like        [Engine](/engine/api-messages/#like-a-message) [Tatcli](/tatcli/tatcli-message/#like-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageLike)
 * list        [Engine](/engine/api-messages/#getting-messages-list) [Tatcli](/tatcli/tatcli-message/#getting-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageList)
 * move        [Engine](/engine/api-messages/#move-a-message-to-another-topic) [Tatcli](/tatcli/tatcli-message/#move-a-message-to-another-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageMove)
 * relabel     [Engine](/engine/api-messages/#remove-all-labels-and-add-new-ones) [Tatcli](/tatcli/tatcli-message/#remove-all-labels-and-add-new-ones-to-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageRelabel)
 * relabelorcreate     [Engine](/engine/api-messages/#remove-all-labels-and-add-new-ones-on-existing-message-create-message-otherwise) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageRelabelOrCreate)
 * reply       [Engine](/engine/api-messages/#reply-to-a-message) [Tatcli](/tatcli/tatcli-message/#reply-to-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageReply)
 * task        [Engine](/engine/api-messages/#create-a-task-from-a-message) [Tatcli](/tatcli/tatcli-message/#create-a-task-from-one-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageTask)
 * unlabel     [Engine](/engine/api-messages/#remove-a-label-from-a-message) [Tatcli](/tatcli/tatcli-message/#remove-a-label-from-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUnlabel)
 * unlike      [Engine](/engine/api-messages/#unlike-a-message) [Tatcli](/tatcli/tatcli-message/#unlike-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUnlike)
 * untask      [Engine](/engine/api-messages/#remove-a-message-from-tasks) [Tatcli](/tatcli/tatcli-message/#remove-a-message-from-tasks) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUntask)
 * unvotedown  [Engine](/engine/api-messages/#remove-vote-down-from-a-message) [Tatcli](/tatcli/tatcli-message/#remove-a-vote-down-from-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUnVoteDown)
 * unvoteup    [Engine](/engine/api-messages/#remove-a-vote-up-from-a-message) [Tatcli](/tatcli/tatcli-message/#remove-a-vote-up-from-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUnVoteUP)
 * update      [Engine](/engine/api-messages/#update-a-message) [Tatcli](/tatcli/tatcli-message/#tatcli-message-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageUpdate)
 * votedown    [Engine](/engine/api-messages/#vote-down-a-message) [Tatcli](/tatcli/tatcli-message/#vote-down-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageVoteDown)
 * voteup      [Engine](/engine/api-messages/#vote-up-a-message) [Tatcli](/tatcli/tatcli-message/#vote-up-a-message) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.MessageVoteUP)
* Group
 * addAdminUser     [Engine](/engine/api-groups/#delete-an-admin-user-from-a-group) [Tatcli](/tatcli/tatcli-group/#tatcli-group-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupAddAdminUsers)
 * addUser          [Engine](/engine/api-groups/#add-a-user-to-a-group) [Tatcli](/tatcli/tatcli-group/#add-user-to-a-group) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupAddUsers)
 * create           [Engine](/engine/api-groups/#create-a-group) [Tatcli](/tatcli/tatcli-group/#create-a-group-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupCreate)
 * delete           [Engine](/engine/api-groups/#delete-a-group) [Tatcli](/tatcli/tatcli-group/#delete-a-group-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupDelete)
 * deleteAdminUser  [Engine](/engine/api-groups/#add-an-admin-user-to-a-group) [Tatcli](/tatcli/tatcli-group/#tatcli-group-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupDeleteAdminUsers)
 * deleteUser       [Engine](/engine/api-groups/#delete-a-user-from-a-group) [Tatcli](/tatcli/tatcli-group/#delete-a-user-from-a-group) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupDeleteUsers)
 * list             [Engine](/engine/api-groups/#getting-groups-list) [Tatcli](/tatcli/tatcli-group/#tatcli-group-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupList)
 * update           [Engine](/engine/api-groups/#update-a-group) [Tatcli](/tatcli/tatcli-group/#update-a-group-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.GroupUpdate)
* User
 * add                           [Engine](/engine/api-users/#create-a-user) [Tatcli](/tatcli/tatcli-user/#create-a-user) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserAdd)
 * addContact                    [Engine](/engine/api-users/#add-a-contact) [Tatcli](/tatcli/tatcli-user/#tatcli-user-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserAddContact)
 * addFavoriteTag                [Engine](/engine/api-users/#add-a-favorite-tag) [Tatcli](/tatcli/tatcli-user/#add-a-favorite-tag) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserAddFavoriteTag)
 * addFavoriteTopic              [Engine](/engine/api-users/#add-a-favorite-topic) [Tatcli](/tatcli/tatcli-user/#add-a-favorite-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserAddFavoriteTopic)
 * archive                       [Engine](/engine/api-users/#archive-a-user) [Tatcli](/tatcli/tatcli-user/#archive-a-user-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserArchive)
 * check                         [Engine](/engine/api-users/#check-private-topics-and-default-group-on-one-user) [Tatcli](/tatcli/tatcli-user/#check-a-user-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserCheck)
 * contacts                      [Engine](/engine/api-users/#get-contacts) [Tatcli](/tatcli/tatcli-user/#get-contacts-presences-since-n-seconds-tatcli-user-contacts-seconds) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserContacts)
 * convert                       [Engine](/engine/api-users/#convert-a-user-to-a-system-user) [Tatcli](/tatcli/tatcli-user/#convert-to-a-system-user-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserConvertToSystem)
 * disableNotificationsAllTopics [Engine](/engine/api-users/#disable-notifications-on-all-topics-except-private) [Tatcli](/tatcli/tatcli-user/#disable-notifications-on-all-topics) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserDisableNotificationsAllTopics)
 * disableNotificationsTopic     [Engine](/engine/api-users/#disable-notifications-on-one-topic) [Tatcli](/tatcli/tatcli-user/#disable-notifications-on-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserDisableNotificationsTopic)
 * enableNotificationsAllTopics  [Engine](/engine/api-users/#enable-notifications-on-all-topics) [Tatcli](/tatcli/tatcli-user/#enable-notifications-on-all-topics) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserEnableNotificationsAllTopics)
 * enableNotificationsTopic      [Engine](/engine/api-users/#enable-notifications-on-one-topic) [Tatcli](/tatcli/tatcli-user/#enable-notifications-on-a-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserEnableNotificationsTopic)
 * list                          [Engine](/engine/api-users/#getting-users-list) [Tatcli](/tatcli/tatcli-user/#list-users) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserList)
 * me                            [Engine](/engine/api-users/#get-information-about-current-user) [Tatcli](/tatcli/tatcli-user/#get-information-about-me) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserMe)
 * removeContact                 [Engine](/engine/api-users/#remove-a-contact) [Tatcli](/tatcli/tatcli-user/#tatcli-user-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserRemoveContact)
 * removeFavoriteTag             [Engine](/engine/api-users/#remove-a-favorite-tag) [Tatcli](/tatcli/tatcli-user/#remove-a-favorite-tag) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserRemoveFavoriteTag)
 * removeFavoriteTopic           [Engine](/engine/api-users/#remove-a-favorite-topic) [Tatcli](/tatcli/tatcli-user/#remove-a-favorite-topic) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserRemoveFavoriteTopic)
 * rename                        [Engine](/engine/api-users/#rename-a-username) [Tatcli](/tatcli/tatcli-user/#rename-a-username-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserRename)
 * reset                         [Engine](/engine/api-users/#ask-for-reset-a-password) [Tatcli](/tatcli/tatcli-user/#ask-for-reset-password) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserReset)
 * resetSystemUser               [Engine](/engine/api-users/#reset-a-password-for-system-user) [Tatcli](/tatcli/tatcli-user/#tatcli-user-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserResetSystem)
 * setAdmin                      [Engine](/engine/api-users/#grant-a-user-to-an-admin-user) [Tatcli](/tatcli/tatcli-user/#grant-a-user-to-tat-admin-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserSetAdmin)
 * update                        [Engine](/engine/api-users/#update-fullname-or-email) [Tatcli](/tatcli/tatcli-user/#update-fullname-and-email-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserUpdate)
 * updateSystemUser              [Engine](/engine/api-users/#update-flags-on-system-user) [Tatcli](/tatcli/tatcli-user/#update-a-system-user-admin-only) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserUpdateSystem)
 * verify                        [Engine](/engine/api-users/#verify-a-user) [Tatcli](/tatcli/tatcli-user/#verify-account) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.UserVerify)
* Presence
 * add                           [Engine](/engine/api-presences/#add-presence) [Tatcli](/tatcli/tatcli-presence/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.PresenceAddAndGet)
 * delete                        [Engine](/engine/api-presences/#delete-presence) [Tatcli](/tatcli/tatcli-presence/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.PresenceDelete)
 * list                          [Engine](/engine/api-presences/#getting-presences) [Tatcli](/tatcli/tatcli-presence/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.PresenceList)
* Stats
 * count              [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsCount)
 * dbCollections      [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBCollections)
 * dbReplSetGetConfig [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBReplSetGetConfig)
 * dbReplSetGetStatus [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBReplSetGetStatus)
 * dbServerStatus     [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBServerStatus)
 * dbSlowestQueries   [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBSlowestQueries)
 * dbstats            [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDBStats)
 * distribution       [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsDistribution)
 * instance           [Engine]() [Tatcli](/tatcli/tatcli-stats/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.StatsInstance)
* System
 * cacheclean   [Engine]() [Tatcli](/tatcli/tatcli-system/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.SystemCacheClean)
 * cacheinfo    [Engine]() [Tatcli](/tatcli/tatcli-system/) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.SystemCacheInfo)
* Version
 * version [Engine]() [Tatcli](/tatcli/tatcli-version/#tatcli-version-h) [Go-SDK](https://godoc.org/github.com/ovh/tat#Client.Version)




## FAQ
*What about attachment (sound, image, etc...) ?*
Tat Engine stores only *text*. Use other application, like Plik (https://github.com/root-gg/plik)
to upload file and store URL on Tat. This workflow should be done by UI.

*In / Out Format*
Tat Engine communicates only with JSON format with its API, even for messages sent above with kafka Hook on a topic.
