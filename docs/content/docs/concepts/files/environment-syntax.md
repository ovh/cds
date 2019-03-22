---
title: "Environment configuration file"
weight: 4
card: 
  name: concept_workflow
  weight: 4
---

## Definition

An environnement is a way to declare and manipulate sets of environment variables and attach them to pipelines within a workflow.
It is also a way to organize your workflow and keep it clear and still readable.

## Format

```yaml
name: MyEnvironment

values:
  myBooleanVariable:
    type: boolean
    value: true

  myNumberVariable:
    type: number
    value: 1

  mySecretVariable:
    type: password
    value: f1a2b3dd756e4db381d7a88631c67355

  myStringVariable:
    value: myStringValue

  myTextVariable:
    type: text
    value: |
      This
      Is a
      multiline text value
      with a project variable inside {{.cds.proj.var}}
```

All variables in CDS have a type: `string` is the default type and can be omitted, `number`, `boolean`, and `text`. You can also define `password`, the value in the file is encrypted. You can generated an encrypted value with the command `cdsctl encrypt MYPROJECT my-data my-super-secret-value`.

All values can references other variables, thanks to the CDS interpolation engine:
```yaml
  myTextVariable:
    type: text
    value: |
      This
      Is a
      multiline text value
      with a project variable inside {{.cds.proj.var}}
```

## File usage

The environment files can be exported and imported from CDS with the following command.

```
➜  ~ cdsctl environment export
```

```
➜  ~ cdsctl environment import
```

The files can also relies in your git repositories if your workflow definition is [stored in your git repository]({{< relref "../../tutorials/init_workflow_with_cdsctl/" >}}).

## Usage in a pipeline

While running a pipeline attached to this environment, you can manipulate those variables in too ways.

 * From the interpolation engine using `{{.cds.env.MyStringVariable}}`
 * From the environment variables using 
  * `CDS_ENV_MYSTRINGVARIABLE`, 
  * `CDS_ENV_MyStringVariable`, 
  * `MYSTRINGVARIABLE`, 
  * `MyStringVariable`

