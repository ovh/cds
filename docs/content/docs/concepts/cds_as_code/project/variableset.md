---
title: "VariableSet"
weight: 4
---

# Description

A variableSet represent a list a variables provided to the jobs during the workflow execution.

A variable can be a:

* `string`
* `secret`

For each type of variable it's possible to provide a JSON value.

# Permission

To be able to manage repository manager you will need the permission `manage` on your project

# Add a variableset using CLI

```
cdsctl experimental project variableset add <PROJECT-KEY> <VARIABLESET-NAME> 
```
* `PROJECT-KEY`: The project key
* `VARIABLESET-NAME`: The name of the variableset

[Full CLI documentation here]({{< relref "/docs/components/cdsctl/experimental/project/variableset/_index.md" >}})

# Add a variable in a variableset using CLI

```
cdsctl experimental project variableset add <PROJECT-KEY> <VARIABLESET-NAME> <NAME> <VALUE> <TYPE> 
```
* `PROJECT-KEY`: The project key
* `VARIABLESET-NAME`: The name of the variableset
* `NAME`: The name of the variable
* `VALUE`: The value of the variable
* `TYPE`: The type of variable:  string | secret

[Full CLI documentation here]({{< relref "/docs/components/cdsctl/experimental/project/variableset/item/_index.md" >}})








