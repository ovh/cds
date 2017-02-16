# Go-Dump

Go-Dump is a package which helps you to dump a struct to `SdtOut`, any `io.Writer`, or a `map[string]string`.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)]
(http://godoc.org/github.com/fsamin/go-dump) [![Build Status](https://travis-ci.org/fsamin/go-dump.svg?branch=master)](https://travis-ci.org/fsamin/go-dump) [![Go Report Card](https://goreportcard.com/badge/github.com/fsamin/go-dump)](https://goreportcard.com/report/github.com/fsamin/go-dump)

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
Pipeline.Name: MyPipeline
Pipeline.Type: build
Pipeline.ProjectID: 0
Pipeline.Stages.Stages0.Stage.ID: 0
Pipeline.Stages.Stages0.Stage.Name: stage 1
Pipeline.Stages.Stages0.Stage.PipelineID: 0
Pipeline.Stages.Stages0.Stage.BuildOrder: 1
Pipeline.Stages.Stages0.Stage.Enabled: true
Pipeline.Stages.Stages0.Stage.LastModified: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.PipelineActionID: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.PipelineStageID: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Enabled: false
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.LastModified: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.ID: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Name: Job 1
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Description: This is job 1
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.ID: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Name: Script
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Type: Builtin
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Parameters.Parameters0.Parameter.ID: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Parameters.Parameters0.Parameter.Name: script
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Parameters.Parameters0.Parameter.Type: text
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Parameters.Parameters0.Parameter.Value: echo lol
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Enabled: false
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.Final: false
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Actions.Actions0.Action.LastModified: 0
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Enabled: false
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.Final: false
Pipeline.Stages.Stages0.Stage.Jobs.Jobs0.Job.Action.Action.LastModified: 0
Pipeline.Permission: 0
Pipeline.LastModified: 0
````

## More examples

See [unit tests](test/dump_test.go) for more examples.
