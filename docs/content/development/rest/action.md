+++
title = "action"
+++


### DELETE `/action/<permActionName>`

URL         | **`/action/<permActionName>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteActionHandler%22)
    









### GET `/action/<actionID>/audit`

URL         | **`/action/<actionID>/audit`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getActionAuditHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getActionAuditHandler%22)
    









### GET `/action/<actionName>/using`

URL         | **`/action/<actionName>/using`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getPipelinesUsingActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPipelinesUsingActionHandler%22)
    









### GET `/action/<permActionName>`

URL         | **`/action/<permActionName>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getActionHandler%22)
    









### PUT `/action/<permActionName>`

URL         | **`/action/<permActionName>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateActionHandler%22)
    









### POST `/action/<permActionName>`

URL         | **`/action/<permActionName>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addActionHandler%22)
    









### GET `/action/<permActionName>/export`

URL         | **`/action/<permActionName>/export`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getActionExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getActionExportHandler%22)
    









### GET `/action/requirement`

URL         | **`/action/requirement`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getActionsRequirements](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getActionsRequirements%22)
    









### List all public actions

URL         | **`/action`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getActionsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getActionsHandler%22)
    









### importAction insert OR update an existing action.

URL         | **`/action/import`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [importActionHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importActionHandler%22)
    









