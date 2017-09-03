# Go-Dump

Go-Dump is a package which helps you to dump a struct to `SdtOut`, any `io.Writer`, or a `map[string]string`.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/fsamin/go-dump) [![Build Status](https://travis-ci.org/fsamin/go-dump.svg?branch=master)](https://travis-ci.org/fsamin/go-dump) [![Go Report Card](https://goreportcard.com/badge/github.com/fsamin/go-dump)](https://goreportcard.com/report/github.com/fsamin/go-dump)

## Sample usage

````golang
type T struct {
    A int
    B string
}

a := T{23, "foo bar"}

dump.FDump(out, a)
````

Will prints

````bash
T.A: 23
T.B: foo bar
````

## Usage with a map

```golang
type T struct {
    A int
    B string
}

a := T{23, "foo bar"}

m, _ := dump.ToMap(a)
```

Will returns such a map:

| KEY           | Value         |
| ------------- | ------------- |
| T.A           | 23            |
| T.B           | foo bar       |

## Formatting keys

```golang
    dump.ToMap(a, dump.WithDefaultLowerCaseFormatter())
```

## Complex example

For the following complex struct:

```golang
    sdk.Pipeline{
            Name: "MyPipeline",
            Type: sdk.BuildPipeline,
            Stages: []sdk.Stage{
                {
                    BuildOrder: 1,
                    Name:       "stage 1",
                    Enabled:    true,
                    Jobs: []sdk.Job{
                        {
                            Action: sdk.Action{
                                Name:        "Job 1",
                                Description: "This is job 1",
                                Actions: []sdk.Action{
                                    {

                                        Type: sdk.BuiltinAction,
                                        Name: sdk.ScriptAction,
                                        Parameters: []sdk.Parameter{
                                            {
                                                Name:  "script",
                                                Type:  sdk.TextParameter,
                                                Value: "echo lol",
                                            },
                                        },
                                    },
                                },
                            },
                        },
                    },
                },
            },
        }
```

Output will be

````bash
Pipeline.AttachedApplication.__Len__: 0
Pipeline.AttachedApplication.__Type__: Array
Pipeline.GroupPermission.__Len__: 0
Pipeline.GroupPermission.__Type__: Array
Pipeline.ID: 0
Pipeline.LastModified: 0
Pipeline.LastPipelineBuild:
Pipeline.Name: MyPipeline
Pipeline.Parameter.__Len__: 0
Pipeline.Parameter.__Type__: Array
Pipeline.Permission: 0
Pipeline.ProjectID: 0
Pipeline.ProjectKey:
Pipeline.Stages.Stages0.BuildOrder: 1
Pipeline.Stages.Stages0.Enabled: true
Pipeline.Stages.Stages0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Actions.__Len__: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Actions.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Description:
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Final: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Name: Script
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Description:
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Name: script
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Type: text
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Value: echo lol
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.__Type__: Parameter
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.__Len__: 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Requirements.__Len__: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Requirements.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Type: Builtin
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.__Type__: Action
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.__Len__: 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Description: This is job 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Final: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Name: Job 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Parameters.__Len__: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Parameters.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Requirements.__Len__: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Requirements.__Type__: Array
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Type:
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.__Type__: Action
Pipeline.Stages.Stages0.Jobs.Jobs0.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.PipelineActionID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.PipelineStageID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.__Type__: Job
Pipeline.Stages.Stages0.Jobs.__Len__: 1
Pipeline.Stages.Stages0.Jobs.__Type__: Array
Pipeline.Stages.Stages0.LastModified: 0
Pipeline.Stages.Stages0.Name: stage 1
Pipeline.Stages.Stages0.PipelineBuildJobs.__Len__: 0
Pipeline.Stages.Stages0.PipelineBuildJobs.__Type__: Array
Pipeline.Stages.Stages0.PipelineID: 0
Pipeline.Stages.Stages0.Prerequisites.__Len__: 0
Pipeline.Stages.Stages0.Prerequisites.__Type__: Array
Pipeline.Stages.Stages0.RunJobs.__Len__: 0
Pipeline.Stages.Stages0.RunJobs.__Type__: Array
Pipeline.Stages.Stages0.Status:
Pipeline.Stages.Stages0.__Type__: Stage
Pipeline.Stages.__Len__: 1
Pipeline.Stages.__Type__: Array
Pipeline.Type: build
__Type__: Pipeline
````

## More examples

See [unit tests](test/dump_test.go) for more examples.

## Dependencies

Go-Dump needs Go >= 1.7

External Dependencies :

- github.com/mitchellh/mapstructure
