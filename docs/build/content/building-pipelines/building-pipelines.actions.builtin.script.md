+++
title = "Script"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "script"

+++

**Script** is a builtin action, you can't modify it.

This action execute a script, written in script attribute

## Parameters

* script: Content of your script. You can put

```bash
#!/bin/bash
```

 or

```bash
#!/bin/perl
```

 at first line.

Make sure that the binary used is in the pre-requisites of action
