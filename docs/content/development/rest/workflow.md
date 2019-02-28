+++
title = "workflow"
+++


### GET `/workflow/artifact/<hash>`

URL         | **`/workflow/artifact/<hash>`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [downloadworkflowArtifactDirectHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+downloadworkflowArtifactDirectHandler%22)
    









### GET `/workflow/hook`

URL         | **`/workflow/hook`**
----------- |----------
Method      | GET     
Permissions |  NeedService:  -  Auth: true
Code        | [getWorkflowHooksHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowHooksHandler%22)
    









### GET `/workflow/hook/model/<model>`

URL         | **`/workflow/hook/model/<model>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowHookModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowHookModelHandler%22)
    









### PUT `/workflow/hook/model/<model>`

URL         | **`/workflow/hook/model/<model>`**
----------- |----------
Method      | PUT     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [putWorkflowHookModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putWorkflowHookModelHandler%22)
    









### POST `/workflow/hook/model/<model>`

URL         | **`/workflow/hook/model/<model>`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postWorkflowHookModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowHookModelHandler%22)
    









### GET `/workflow/outgoinghook/model`

URL         | **`/workflow/outgoinghook/model`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowOutgoingHookModelsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowOutgoingHookModelsHandler%22)
    









