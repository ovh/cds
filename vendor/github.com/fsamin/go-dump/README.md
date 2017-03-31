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
Pipeline.LastModified: 0
Pipeline.Name: MyPipeline
Pipeline.Permission: 0
Pipeline.ProjectID: 0
Pipeline.Stages.Stages0.BuildOrder: 1
Pipeline.Stages.Stages0.Enabled: true
Pipeline.Stages.Stages0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Final: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Name: Script
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Name: script
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Type: text
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Parameters.Parameters0.Value: echo lol
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Actions.Actions0.Type: Builtin
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Description: This is job 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Final: false
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.ID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.Action.Name: Job 1
Pipeline.Stages.Stages0.Jobs.Jobs0.Enabled: false
Pipeline.Stages.Stages0.Jobs.Jobs0.LastModified: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.PipelineActionID: 0
Pipeline.Stages.Stages0.Jobs.Jobs0.PipelineStageID: 0
Pipeline.Stages.Stages0.LastModified: 0
Pipeline.Stages.Stages0.Name: stage 1
Pipeline.Stages.Stages0.PipelineID: 0
````

## More examples

See [unit tests](test/dump_test.go) for more examples.

## Dependencies

Go-Dump needs Go >= 1.7

External Dependencies :

- github.com/mitchellh/mapstructure