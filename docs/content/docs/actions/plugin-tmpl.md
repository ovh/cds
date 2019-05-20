---
title: "plugin-tmpl"
card:
  name: plugin
---

This action helps you generates a file using a template file and text/template golang package.

Check documentation on text/template for more information https://golang.org/pkg/text/template.


## Parameters

* **file**: Template file to use
* **output**: Output path for generated file (default to <file>.out or just trimming .tpl extension)
* **params**: Parameters to pass on the template file (key=value newline separated list)



