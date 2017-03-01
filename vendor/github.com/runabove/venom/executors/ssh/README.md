# Venom - Executor SSH

Step for execute a script on remote server via ssh


## Input

In your yaml file, you can use:

```yaml
  - host mandatory
  - command mandatory
  - user optional (default is os username)
  - password optional (mandatory if no privatekey is found)
  - privatekey optional (default is $HOME/.ssh/id_rsa)
```

Example

```yaml

name: Title of TestSuite
testcases:
- name: Check if exit code != 1 and echo command response in less than 1s
  steps:
  - type: ssh
    host: localhost:2222
    command: echo 'foo'
    assertions:
    - result.code ShouldEqual 0
    - result.timeseconds ShouldBeLessThan 1

```

## Output

```yaml
executor
systemout
systemerr
err
code
timeseconds
timehuman
```

- result.timeseconds & result.timehuman: time of execution
- result.executor.executor.script: script executed
- result.err: if exists, this field contains error
- result.systemout: Standard Output of executed script
- result.systemerr: Error Output of executed script
- result.code: Exit Code

## Default assertion

```yaml
result.code ShouldEqual 0
```
