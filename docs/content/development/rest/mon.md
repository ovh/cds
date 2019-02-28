+++
title = "mon"
+++


### GET `/mon/db/migrate`

URL         | **`/mon/db/migrate`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getMonDBStatusMigrateHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getMonDBStatusMigrateHandler%22)
    









### GET `/mon/errors/<uuid>`

URL         | **`/mon/errors/<uuid>`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getErrorHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getErrorHandler%22)
    









### GET `/mon/panic/<uuid>`

URL         | **`/mon/panic/<uuid>`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getPanicDumpHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getPanicDumpHandler%22)
    









### GET `/mon/smtp/ping`

URL         | **`/mon/smtp/ping`**
----------- |----------
Method      | GET     
Permissions |  Auth: true -  Auth: true
Code        | [smtpPingHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+smtpPingHandler%22)
    









### GET `/mon/status`

URL         | **`/mon/status`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [statusHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+statusHandler%22)
    









