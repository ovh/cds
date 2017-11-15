+++
title = "Release"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "release"

+++

**Release** is a builtin action, you can't modify it.

This action creates a release on the git repository linked to the application (if repository manager implements it.).

## Parameters

* artifacts - optional - List of artifacts to upload, separated by ','. You can also use regexp
* releaseNote - optional - release information
* tag - mandatory - Tag attached to the release
* title - mandatory - Set the title of the release

### Example

Coming soon