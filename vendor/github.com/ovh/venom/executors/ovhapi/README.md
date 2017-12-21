# Venom - Executor OVHAPI

Step to test ovh api

Use case: you software need to make call to OVH api.<br>
You will need ovh credentials to make api call. Please follow this tutorial to get all needed keys : <br>
FR : https://www.ovh.com/fr/g934.premiers_pas_avec_lapi <br>
EN : https://api.ovh.com/g934.first_step_with_api

## Input
In your yaml file, you can use:

```
  - method optional, default value : GET
  - path mandatory, example "/me"
  - noAuth optional
  - body optional
  - bodyFile optional
```

```yaml

name: Title of TestSuite
testcases:
- name: me
  context:
    type: default
    endpoint: 'ovh-eu'
    applicationKey: 'APPLICATION_KEY'
    applicationSecret: 'APPLICATION_SECRET'
    consumerKey: 'CONSUMER_KEY'
    insecureTLS: true #default false
  steps:
  - type: ovhapi
    method: GET
    path: /me
    headers:
      header1: value1
      header2: value2
    retry: 3
    delay: 2
    assertions:
    - result.statuscode ShouldEqual 200
    - result.bodyjson.nichandle ShouldContainSubstring MY_NICHANDLE

```

## Output

```
result.executor
result.timeseconds
result.timehuman
result.statuscode
result.body
result.bodyjson
result.error
```
- result.timeseconds & result.timehuman: time of execution
- result.err: if exists, this field contains error
- result.body: body of HTTP response
- result.bodyjson: body of HTTP response if it's a json. You can access json data as result.bodyjson.yourkey for example
- result.statuscode: Status Code of HTTP response

## Default assertion

```yaml
result.statuscode ShouldEqual 200
```
