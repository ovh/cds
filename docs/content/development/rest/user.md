+++
title = "user"
+++


### AddUser creates a new user and generate verification email

URL         | **`/user/signup`**
----------- |----------
Method      | POST     
Permissions |  Auth: false
Code        | [addUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+addUserHandler%22)
    









### ConfirmUser verify token send via email and mark user as verified

URL         | **`/user/<username>/confirm/<token>`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [confirmUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+confirmUserHandler%22)
    









### DeleteUser removes a user

URL         | **`/user/<username>`**
----------- |----------
Method      | DELETE     
Permissions |  NeedUsernameOrAdmin: true -  Auth: true
Code        | [deleteUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+deleteUserHandler%22)
    









### GET `/user/timeline/filter`

URL         | **`/user/timeline/filter`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getTimelineFilterHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getTimelineFilterHandler%22)
    









### GET `/user/timeline`

URL         | **`/user/timeline`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getTimelineHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getTimelineHandler%22)
    









### GET `/user/token/<token>`

URL         | **`/user/token/<token>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getUserTokenHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUserTokenHandler%22)
    









### GET `/user/token`

URL         | **`/user/token`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getUserTokenListHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUserTokenListHandler%22)
    









### GetUser returns a specific user's information

URL         | **`/user/<username>`**
----------- |----------
Method      | GET     
Permissions |  NeedUsernameOrAdmin: true -  Auth: true
Code        | [getUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUserHandler%22)
    









### GetUsers fetches all users from databases

URL         | **`/user`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getUsersHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUsersHandler%22)
    









### POST `/user/<username>/reset`

URL         | **`/user/<username>/reset`**
----------- |----------
Method      | POST     
Permissions |  Auth: false
Code        | [resetUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+resetUserHandler%22)
    









### POST `/user/import`

URL         | **`/user/import`**
----------- |----------
Method      | POST     
Permissions |  NeedAdmin: true -  Auth: true
Code        | [importUsersHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+importUsersHandler%22)
    









### POST `/user/timeline/filter`

URL         | **`/user/timeline/filter`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postTimelineFilterHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postTimelineFilterHandler%22)
    









### UpdateUser modifies user informations

URL         | **`/user/<username>`**
----------- |----------
Method      | PUT     
Permissions |  NeedUsernameOrAdmin: true -  Auth: true
Code        | [updateUserHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+updateUserHandler%22)
    









### getUserGroups returns groups of the user

URL         | **`/user/<username>/groups`**
----------- |----------
Method      | GET     
Permissions |  NeedUsernameOrAdmin: true -  Auth: true
Code        | [getUserGroupsHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUserGroupsHandler%22)
    









### getUserLogged check if the current user is connected

URL         | **`/user/me`**
----------- |----------
Method      | GET     
Permissions |  Auth: false
Code        | [getUserLoggedHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getUserLoggedHandler%22)
    









### postUserFavorite post favorite user for workflow or project

URL         | **`/user/favorite`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postUserFavoriteHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postUserFavoriteHandler%22)
    









