+++
draft = false
title = "Configuration"
description = ""

[menu.main]
parent = "start"
identifier = "configuration"
weight = 2

+++

When building the website, you can set a theme by using `--theme` option. We suggest you to edit your configuration file and set the theme by default. Example with `config.toml` format.

```
theme = "docdock"
```

## Search index generation

Add the follow line in the same `config.toml` file.

```
[outputs]
home = [ "HTML", "RSS", "JSON"]
```

LUNRJS search index file will be generated on content changes.

## Your website's content

Find out how to [create]({{%relref "page.md"%}}) and [organize your content]({{%relref "organisation.md"%}}) quickly and intuitively.