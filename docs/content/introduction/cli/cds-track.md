+++
title = "Track CDS Pipeline"
weight = 7

[menu.main]
parent = "cli"
identifier = "cli-git-track"

+++

### Introduction

This tutorial introduce the `cds track <git commit>` function of CDS [cli](/cli).

## Goal: Immediate feedback

cds track aims to display in your terminal the status of the pipeline building code refered by given commit hash.

Push your branch, start cds track and get immediate feedback.

Git track will display all pipelines related to given hash.

This means triggered testing and deployment pipelines will be displayed.

![git-track](/images/tutorials_git_track.png)

## Git alias sugar

To enhance even more your daily routine, you can create a git alias:

```bash
$ cat ~/.gitconfig
[alias]
  track = !cds -w track $(git rev-parse HEAD)
```
