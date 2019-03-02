+++
title = "group"
+++


### DELETE `/group/<group-name>/token/<tokenid>`

URL         | **`/group/<group-name>/token/<tokenid>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteTokenHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteTokenHandler%22)
    









### DELETE `/group/<group-name>/user/<user-name>/admin`

URL         | **`/group/<group-name>/user/<user-name>/admin`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [removeUserGroupAdminHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+removeUserGroupAdminHandler%22)
    









### DELETE `/group/<group-name>/user/<user-name>`

URL         | **`/group/<group-name>/user/<user-name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [removeUserFromGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+removeUserFromGroupHandler%22)
    









### DELETE `/group/<group-name>`

URL         | **`/group/<group-name>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteGroupHandler%22)
    









### GET `/group/<group-name>/token`

URL         | **`/group/<group-name>/token`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getGroupTokenListHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGroupTokenListHandler%22)
    









### GET `/group/<group-name>`

URL         | **`/group/<group-name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGroupHandler%22)
    









### GET `/group/public`

URL         | **`/group/public`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getPublicGroupsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPublicGroupsHandler%22)
    









### GET `/group`

URL         | **`/group`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getGroupsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGroupsHandler%22)
    









### POST `/group/<group-name>/user/<user-name>/admin`

URL         | **`/group/<group-name>/user/<user-name>/admin`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [setUserGroupAdminHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+setUserGroupAdminHandler%22)
    









### POST `/group/<group-name>/user`

URL         | **`/group/<group-name>/user`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addUserInGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addUserInGroupHandler%22)
    









### POST `/group`

URL         | **`/group`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addGroupHandler%22)
    









### PUT `/group/<group-name>`

URL         | **`/group/<group-name>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateGroupHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateGroupHandler%22)
    









### generateToken allows a user to generate a token associated to a group permission

URL         | **`/group/<group-name>/token`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [generateTokenHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+generateTokenHandler%22)
    









