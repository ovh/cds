# Venom - Executor HTTP

Step for sending SMTP

Use case: you software have to check mails for doing something with them ?
Venom can send mail then execute some tests on your software.

## Input
In your yaml file, you can use:

```yaml
  - method optional, default value : GET
  - url mandatory
  - path optional
  - body optional
  - bodyFile optional
  - headers optional
```

```yaml

name: Title of TestSuite
testcases:

- name: GET http testcase
  steps:
  - type: http
    method: GET
    url: https://eu.api.ovh.com/1.0/
    assertions:
    - result.body ShouldContainSubstring /dedicated/server
    - result.body ShouldContainSubstring /ipLoadbalancing
    - result.statuscode ShouldEqual 200
    - result.bodyjson.apis.apis0.path ShouldEqual /allDom


- name: POST http with bodyFile
  steps:
  - type: http
    method: POST
    url: https://eu.api.ovh.com/1.0/
    bodyFile: /tmp/myfile.tmp
    assertions:
    - result.statuscode ShouldNotEqual 200


- name: POST http with multipart
  steps:
  - type: http
    method: POST
    url: https://eu.api.ovh.com/1.0/
    multipart_form:
        file: '@/tmp/myfile.tmp'
    assertions:
    - result.statuscode ShouldNotEqual 200
```
*NB: to post a file with multipart_form, prefix the path to the file with '@'*

## Output

```
result.executor
result.timeseconds
result.timehuman
result.statuscode
result.body
result.bodyjson
result.headers
result.error
```
- result.timeseconds & result.timehuman: time of execution
- result.executor.executor.method: HTTP method used, example: GET
- result.executor.executor.url: url called
- result.executor.executor.multipartform: multipartform if exists
- result.err: if exists, this field contains error
- result.body: body of HTTP response
- result.bodyjson: body of HTTP response if it's a json. You can access json data as result.bodyjson.yourkey for example
- result.headers: headers of HTTP response
- result.statuscode: Status Code of HTTP response

## Default assertion

```yaml
result.statuscode ShouldEqual 200
```
