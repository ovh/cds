---
title: "plugin-group-tmpl"
card:
  name: plugin
---

This actions helps you generate a marathon group application file.
It takes a config template file as a single application, and creates the group with the variables specified for each application in the applications files.
Check documentation on text/template for more information https://golang.org/pkg/text/template.


## Parameters

* **applications**: Applications file variables
* **config**: Template file to use
* **output**: Output path for generated file (default to <file>.out or just trimming .tpl extension)



