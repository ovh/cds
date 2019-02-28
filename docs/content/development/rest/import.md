+++
title = "import"
+++


### Import workflow as code

URL         | **`/import/<project-key>`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postImportAsCodeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postImportAsCodeHandler%22)
    

#### Description
This the entrypoint to perform workflow as code. The first step is to post an operation leading to checkout application and scrapping files


#### Request Body
```
{"vcs_Server":"github", "url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
```

#### Response Body
```
{"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"https","ssh_key":"","user":"","password":"","branch":"","default_branch":"master","pgp_key":""},"setup":{"checkout":{"branch":"master"}}}
```


### Get import workflow as code operation details

URL         | **`/import/<project-key>/<uuid>`**
----------- |----------
Method      | GET     
Permissions |  Auth: true
Code        | [getImportAsCodeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+getImportAsCodeHandler%22)
    

#### Description
This route helps you to know if a "import as code" is over, and the details of the performed operation


#### Request Body
```
None
```

#### Response Body
```
{"uuid":"ee3946ac-3a77-46b1-af78-77868fde75ec","url":"https://github.com/fsamin/go-repo.git","strategy":{"connection_type":"","ssh_key":"","user":"","password":"","branch":"","default_branch":"","pgp_key":""},"setup":{"checkout":{}},"load_files":{"pattern":".cds/**/*.yml","results":{"w-go-repo.yml":"bmFtZTogdy1nby1yZXBvCgkJCQkJdmVyc2lvbjogdjEuMAoJCQkJCXBpcGVsaW5lOiBidWlsZAoJCQkJCWFwcGxpY2F0aW9uOiBnby1yZXBvCgkJCQkJcGlwZWxpbmVfaG9va3M6CgkJCQkJLSB0eXBlOiBSZXBvc2l0b3J5V2ViSG9vawoJCQkJCQ=="}},"status":2}
```


### Perform workflow as code import

URL         | **`/import/<project-key>/<uuid>/perform`**
----------- |----------
Method      | POST     
Permissions |  Auth: true
Code        | [postPerformImportAsCodeHandler](https://github.com/ovh/cds/search?q=%22func+%28api+*API%29+postPerformImportAsCodeHandler%22)
    

#### Description
This operation push the workflow as code into the project


#### Request Body
```
None
```

#### Response Body
```
translated message list
```


