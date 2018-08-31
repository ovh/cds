# Venom - Executor Read file

Step for Read file

Use case: you software write a file. Venom checks that file is produced, read it,
and return content. Content can be used by another steps of testsuite.

Path can contains a fullpath, a wildcard or a directory:

```
- path: /a/full/path/file.txt
- path: afile.txt
- path: *.yml
- path: a_directory/
- path: ./foo/b*/**/z*.txt
```

## Input

```yaml
name: TestSuite Read File
testcases:
- name: TestCase Read File
  steps:
  - type: readfile
    path: yourfile.txt
    assertions:
    - result.err ShouldNotExist
```

## Output

```yaml
  result.timeseconds
  result.timehuman
  result.executor.executor.path
  result.content
  result.contentjson
  result.size.filename
  result.md5sum.filename
  result.modtime.filename
  result.mod.filename
```

- result.timeseconds & result.timehuman: time for read file
- result.executor.path: executor condition with file path
- result.err: if exist, this field contains error
- result.content: content of readed file
- result.contentjson: content of readed file if it's a json. You can access json data as result.contentjson.yourkey for example
- result.size.filename: size of file 'filename'
- result.md5sum.filename: md5 of file 'fifename'
- result.modtime.filename: date modification of file 'filename', example: 1487698253
- result.mod.filename: rights on file 'filename', example: -rw-r--r--

## Default assertion

```yaml
result.err ShouldNotExist
```

## Example

testa.txt file:

```
simple content
multilines
```

testa.json file:

```json
{
  "foo": "bar"
}
```

testb.json file:

```json
[
  {
    "foo": "bar",
    "foo2": "bar2"
  }
]

```

venom test file:

```yaml
name: TestSuite Read File
testcases:
- name: TestCase Read File
  steps:
  - type: readfile
    path: testa.json
    assertions:
      - result.contentjson.foo ShouldEqual bar

  - type: readfile
    path: testb.json
    assertions:
      - result.contentjson.contentjson0.foo2 ShouldEqual bar2

  - type: readfile
    path: test.txt
    assertions:
      - result.content ShouldContainSubstring multilines
```
