# Venom - Executor Exec

Step for execute a script


## Input

Example

```yaml

name: Title of TestSuite
testcases:
- name: Check if exit code != 1 and echo command response in less than 1s
  steps:
  - script: echo 'foo'
    assertions:
    - result.code ShouldEqual 0
    - result.timeseconds ShouldBeLessThan 1

```

Multiline script:

```yaml
name: Title of TestSuite
testcases:
- name: multiline script
  steps:
  - script: |
            echo "Foo" \
            echo "Bar"
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
