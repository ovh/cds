+++
title = "admin"
+++


### DELETE `/admin/database/migration/delete/<id>`

URL         | **`/admin/database/migration/delete/<id>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteDatabaseMigrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteDatabaseMigrationHandler%22)
    









### DELETE `/admin/plugin/<name>/binary/<os>/<arch>`

URL         | **`/admin/plugin/<name>/binary/<os>/<arch>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteGRPCluginBinaryHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteGRPCluginBinaryHandler%22)
    









### DELETE `/admin/plugin/<name>`

URL         | **`/admin/plugin/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteGRPCluginHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteGRPCluginHandler%22)
    









### DELETE `/admin/service/<name>`

URL         | **`/admin/service/<name>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteAdminServiceHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteAdminServiceHandler%22)
    









### DELETE `/admin/services/call`

URL         | **`/admin/services/call`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [deleteAdminServiceCallHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteAdminServiceCallHandler%22)
    









### DELETE `/admin/warning`

URL         | **`/admin/warning`**
----------- |----------
Method      | DELETE     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [adminTruncateWarningsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+adminTruncateWarningsHandler%22)
    









### GET `/admin/cds/migration`

URL         | **`/admin/cds/migration`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getAdminMigrationsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAdminMigrationsHandler%22)
    









### GET `/admin/database/migration`

URL         | **`/admin/database/migration`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getDatabaseMigrationHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getDatabaseMigrationHandler%22)
    









### GET `/admin/plugin/<name>/binary/<os>/<arch>/infos`

URL         | **`/admin/plugin/<name>/binary/<os>/<arch>/infos`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getGRPCluginBinaryInfosHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGRPCluginBinaryInfosHandler%22)
    









### GET `/admin/plugin/<name>/binary/<os>/<arch>`

URL         | **`/admin/plugin/<name>/binary/<os>/<arch>`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getGRPCluginBinaryHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGRPCluginBinaryHandler%22)
    









### GET `/admin/plugin/<name>`

URL         | **`/admin/plugin/<name>`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getGRPCluginHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getGRPCluginHandler%22)
    









### GET `/admin/plugin`

URL         | **`/admin/plugin`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getAllGRPCluginHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAllGRPCluginHandler%22)
    









### GET `/admin/service/<name>`

URL         | **`/admin/service/<name>`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getAdminServiceHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAdminServiceHandler%22)
    









### GET `/admin/services/call`

URL         | **`/admin/services/call`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getAdminServiceCallHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAdminServiceCallHandler%22)
    









### GET `/admin/services`

URL         | **`/admin/services`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getAdminServicesHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getAdminServicesHandler%22)
    









### POST `/admin/cds/migration/<id>/cancel`

URL         | **`/admin/cds/migration/<id>/cancel`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postAdminMigrationCancelHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postAdminMigrationCancelHandler%22)
    









### POST `/admin/cds/migration/<id>/todo`

URL         | **`/admin/cds/migration/<id>/todo`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postAdminMigrationTodoHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postAdminMigrationTodoHandler%22)
    









### POST `/admin/database/migration/unlock/<id>`

URL         | **`/admin/database/migration/unlock/<id>`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postDatabaseMigrationUnlockedHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postDatabaseMigrationUnlockedHandler%22)
    









### POST `/admin/maintenance`

URL         | **`/admin/maintenance`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postMaintenanceHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postMaintenanceHandler%22)
    









### POST `/admin/plugin/<name>/binary`

URL         | **`/admin/plugin/<name>/binary`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postGRPCluginBinaryHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postGRPCluginBinaryHandler%22)
    









### POST `/admin/plugin`

URL         | **`/admin/plugin`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postPGRPCluginHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPGRPCluginHandler%22)
    









### POST `/admin/services/call`

URL         | **`/admin/services/call`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [postAdminServiceCallHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postAdminServiceCallHandler%22)
    









### PUT `/admin/plugin/<name>`

URL         | **`/admin/plugin/<name>`**
----------- |----------
Method      | PUT     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [putGRPCluginHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putGRPCluginHandler%22)
    









### PUT `/admin/services/call`

URL         | **`/admin/services/call`**
----------- |----------
Method      | PUT     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [putAdminServiceCallHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+putAdminServiceCallHandler%22)
    









### getCPUProfile responds with the pprof-formatted cpu profile.

URL         | **`/admin/debug/cpu`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getCPUProfileHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getCPUProfileHandler%22)
    









### getProfile responds with the pprof-formatted profile named by the request.

URL         | **`/admin/debug/<name>`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getProfileHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProfileHandler%22)
    









### getProfileIndex returns the profiles index

URL         | **`/admin/debug`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getProfileIndexHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getProfileIndexHandler%22)
    









### getTrace responds with the execution trace in binary form.

URL         | **`/admin/debug/trace`**
----------- |----------
Method      | GET     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [getTraceHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getTraceHandler%22)
    









