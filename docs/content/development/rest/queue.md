+++
title = "queue"
+++


### DELETE `/queue/workflows/<id>/book`

URL         | **`/queue/workflows/<id>/book`**
----------- |----------
Method      | DELETE     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [deleteBookWorkflowJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteBookWorkflowJobHandler%22)
    









### GET `/queue/workflows/<id>/infos`

URL         | **`/queue/workflows/<id>/infos`**
----------- |----------
Method      | GET     
Permissions |  NeedWorker:  -  NeedHatchery:  -  Auth: true
Code        | [getWorkflowJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowJobHandler%22)
    









### GET `/queue/workflows/count`

URL         | **`/queue/workflows/count`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [countWorkflowJobQueueHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+countWorkflowJobQueueHandler%22)
    









### GET `/queue/workflows`

URL         | **`/queue/workflows`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getWorkflowJobQueueHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getWorkflowJobQueueHandler%22)
    









### POST `/queue/workflows/<id>/attempt`

URL         | **`/queue/workflows/<id>/attempt`**
----------- |----------
Method      | POST     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [postIncWorkflowJobAttemptHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postIncWorkflowJobAttemptHandler%22)
    









### POST `/queue/workflows/<id>/book`

URL         | **`/queue/workflows/<id>/book`**
----------- |----------
Method      | POST     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [postBookWorkflowJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postBookWorkflowJobHandler%22)
    









### POST `/queue/workflows/<id>/spawn/infos`

URL         | **`/queue/workflows/<id>/spawn/infos`**
----------- |----------
Method      | POST     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [postSpawnInfosWorkflowJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postSpawnInfosWorkflowJobHandler%22)
    









### POST `/queue/workflows/<id>/take`

URL         | **`/queue/workflows/<id>/take`**
----------- |----------
Method      | POST     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postTakeWorkflowJobHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postTakeWorkflowJobHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/coverage`

URL         | **`/queue/workflows/<token>/coverage`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobCoverageResultsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobCoverageResultsHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/log`

URL         | **`/queue/workflows/<token>/log`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobLogsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobLogsHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/result`

URL         | **`/queue/workflows/<token>/result`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobResultHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobResultHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/step`

URL         | **`/queue/workflows/<token>/step`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobStepStatusHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobStepStatusHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/tag`

URL         | **`/queue/workflows/<token>/tag`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobTagsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobTagsHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/test`

URL         | **`/queue/workflows/<token>/test`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobTestsResultsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobTestsResultsHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/variable`

URL         | **`/queue/workflows/<token>/variable`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postWorkflowJobVariableHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobVariableHandler%22)
    









### POSTEXECUTE `/queue/workflows/<token>/vulnerability`

URL         | **`/queue/workflows/<token>/vulnerability`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedWorker:  -  Auth: true
Code        | [postVulnerabilityReportHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postVulnerabilityReportHandler%22)
    









### POSTEXECUTE `/queue/workflows/log/service`

URL         | **`/queue/workflows/log/service`**
----------- |----------
Method      | POSTEXECUTE     
Permissions |  NeedHatchery:  -  Auth: true
Code        | [postWorkflowJobServiceLogsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postWorkflowJobServiceLogsHandler%22)
    









