+++
title = "worker"
+++


### DELETE `/worker/model/<permModelID>`

URL         | **`/worker/model/<permModelID>`**
----------- |----------
Method      | DELETE     
Permissions |  Auth: true
Code        | [deleteWorkerModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteWorkerModelHandler%22)
    









### DELETE `/worker/model/pattern/<type>/<name>`

URL         | **`/worker/model/pattern/<type>/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteWorkerModelPatternHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteWorkerModelPatternHandler%22)
    









### GET `/worker/model/<modelID>/usage`

URL         | **`/worker/model/<modelID>/usage`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelUsageHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelUsageHandler%22)
    









### GET `/worker/model/<permModelID>/export`

URL         | **`/worker/model/<permModelID>/export`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelExportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelExportHandler%22)
    









### GET `/worker/model/capability/type`

URL         | **`/worker/model/capability/type`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getRequirementTypesHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getRequirementTypesHandler%22)
    









### GET `/worker/model/communication`

URL         | **`/worker/model/communication`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelCommunicationsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelCommunicationsHandler%22)
    









### GET `/worker/model/enabled`

URL         | **`/worker/model/enabled`**
----------- |----------
Method      | GET     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [getWorkerModelsEnabledHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelsEnabledHandler%22)
    









### GET `/worker/model/pattern/<type>/<name>`

URL         | **`/worker/model/pattern/<type>/<name>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelPatternHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelPatternHandler%22)
    









### GET `/worker/model/pattern`

URL         | **`/worker/model/pattern`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelPatternsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelPatternsHandler%22)
    









### GET `/worker/model/type`

URL         | **`/worker/model/type`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelTypesHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelTypesHandler%22)
    









### GET `/worker/model`

URL         | **`/worker/model`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkerModelsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkerModelsHandler%22)
    









### GET `/worker`

URL         | **`/worker`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkersHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkersHandler%22)
    









### POST `/worker/<id>/disable`

URL         | **`/worker/<id>/disable`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [disableWorkerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+disableWorkerHandler%22)
    









### POST `/worker/checking`

URL         | **`/worker/checking`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [workerCheckingHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+workerCheckingHandler%22)
    









### POST `/worker/model/pattern`

URL         | **`/worker/model/pattern`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postAddWorkerModelPatternHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postAddWorkerModelPatternHandler%22)
    









### POST `/worker/model`

URL         | **`/worker/model`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [addWorkerModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addWorkerModelHandler%22)
    









### POST `/worker/refresh`

URL         | **`/worker/refresh`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [refreshWorkerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+refreshWorkerHandler%22)
    









### POST `/worker/unregister`

URL         | **`/worker/unregister`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [unregisterWorkerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+unregisterWorkerHandler%22)
    









### POST `/worker/waiting`

URL         | **`/worker/waiting`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [workerWaitingHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+workerWaitingHandler%22)
    









### POST `/worker`

URL         | **`/worker`**
----------- |----------
Method      | POST     
Permissions |  Auth: false
Code        | [registerWorkerHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+registerWorkerHandler%22)
    









### PUT `/worker/model/<permModelID>`

URL         | **`/worker/model/<permModelID>`**
----------- |----------
Method      | PUT     
Permissions |  Auth: true
Code        | [updateWorkerModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateWorkerModelHandler%22)
    









### PUT `/worker/model/book/<permModelID>`

URL         | **`/worker/model/book/<permModelID>`**
----------- |----------
Method      | PUT     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [bookWorkerModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+bookWorkerModelHandler%22)
    









### PUT `/worker/model/error/<permModelID>`

URL         | **`/worker/model/error/<permModelID>`**
----------- |----------
Method      | PUT     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [spawnErrorWorkerModelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+spawnErrorWorkerModelHandler%22)
    









### PUT `/worker/model/pattern/<type>/<name>`

URL         | **`/worker/model/pattern/<type>/<name>`**
----------- |----------
Method      | PUT     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [putWorkerModelPatternHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putWorkerModelPatternHandler%22)
    









### import a worker model yml/json file

URL         | **`/worker/model/import`**
----------- |----------
Method      | POST     
Query Parameter | force=true or false. If false and if the worker model already exists, raise an error
Permissions |  Auth: true
Code        | [postWorkerModelImportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkerModelImportHandler%22)
    

#### Description
import a worker model yml/json file with `cdsctl worker model import mywm.yml`







