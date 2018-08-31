# Venom - Executor Redis

Step to execute command into redis

Use case: your software need to make call to a redis.


## Input

In your yaml file, you can use:

- context : required. Must contains dialURL property. DialURL connects to a Redis server at the given URL using the Redis URI scheme.
URLs should follow the draft IANA specification for the scheme (https://www.iana.org/assignments/uri-schemes/prov/redis).
- commands : an array of redis command
- path : a file which contains a series of redis command. See example below.

If path property is filled, commands property will be ignored.

```yaml
name: My Redis testsuite
testcases:
- name: Commands_Test_Case
  context:
    dialURL: "redis://user:secret@localhost:6379/0"
    type: redis
  steps:
  - type: redis
    commands:
        - FLUSHALL
  - type: redis
    commands:
        - SET foo bar
        - GET foo
        - KEYS *
    assertions:
        - result.commands.commands0.response ShouldEqual OK
        - result.commands.commands1.response ShouldEqual bar
        - result.commands.commands2.response.response0 ShouldEqual foo
  - type: redis
    commands:
        - KEYS *
    assertions:
        - result.commands.commands0.response.response0 ShouldEqual foo
- name: File_Test_Case
  context:
      dialURL: "redis://localhost:6379/0"
      type: redis
  steps:
  - type: redis
    commands:
        - FLUSHALL
  - type: redis
    path: testredis/commands.txt
    assertions:
        - result.commands.commands0.response ShouldEqual OK
        - result.commands.commands1.response ShouldEqual bar
        - result.commands.commands2.response.response0 ShouldEqual foo

```

File is read line by line and each command is split by [strings.Fields](https://golang.org/pkg/strings/#Fields) method

```text
SET Foo Bar
SET bar beez
SET Bar {"foo" : "bar", "poo" : ["lol", "lil", "greez"]}
Keys *
```




## Output

```
result.executor
result.commands
```

- result.executor.commands contains the list of redis command
- result.executor.FilePath contains the path of file

- result.commands contains the list of executed redis command
- result.commands.commandI.response represents the response of redis command. It can be an array or a string, depends of redis command