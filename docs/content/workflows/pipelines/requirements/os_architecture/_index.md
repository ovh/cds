+++
title = "OS & Architecture"
weight = 1

+++

The OS-Architecture prerequisiste allow you to require a worker with a specific OS & Architecture.

**Beware about default value**: there is a default value for OS & Architecture, it's specified in CDS API Configuration.

If user does not specify a prerequisite `os-architecture`, the default value is applied when the job is in CDS Queue.

Then, a hatchery will spawn a worker compiled with the specified `os-architecture` prerequisite.

**Bewere about launching job**: if you put a prerequisite `os-architecture` with value `linux/386`, the job won't be launched by a worker `linux/amd64` even if technically speaking, the worker could launch this job without issue.

### How to set a OS & Architecture

![Step](/images/workflows.pipelines.requirements.os_architecture.choose.png)

### Setup default OS & Architecture on a CDS API Configuration

```toml
#####################
# API Configuration
#####################
[api]

  # if no model and no os/arch is specified in your job's requirements then spawn worker on this architecture (example: amd64, arm, 386)
  defaultArch = "amd64"

  # if no model and os/arch is specified in your job's requirements then spawn worker on this operating system (example: freebsd, linux, windows)
  defaultOS = "linux"
```