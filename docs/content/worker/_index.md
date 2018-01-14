+++
title = "Worker"
weight = 4

+++


### Introduction

#### Why would you need to setup your own worker ?

There is several cases where one would need to setup his own worker:

 * Perform incremental build
 * Build on a specific architecture
 * Perform integration tests in a specific network

### How does this work ?

Workers authenticate on CDS with a token and have the same permissions as the user who generated it.

Generate a token with the following cdsctl command:

```bash
$ cdsctl generate token -g yourgroup -e persistent
```

### Linux Setup

#### Download the binary

Simple enough, run

```bash
$ wget -nv https://your-cds-api/download/worker/`uname -m` -O worker
```

or download from https://github.com/ovh/cds/releases

#### Startup the worker

Mandatory parameters are **--api** and **--token**.

The most basic way to start a worker is as following:

```
$ worker --api=https://your-cds-api --token=NTU2ZmFiOGZmMzI5MGU1NzVmY2FhNThmOTY3NjFmMDVmNmIxOTFhNDViNjRjETCETC
2016/03/24 11:30:50 [NOTICE]   What a good time to be alive
2016/03/24 11:30:50 [NOTICE]   Disconnected from CDS engine, trying to register...
2016/03/24 11:30:50 [NOTICE]   Registering [desk32345] at [https://your-cds-api]
```

That's it, you are done here.

### Windows Setup
#### Download the binary
Download the windows binary from https://github.com/ovh/cds/releases

#### Prerequisites
CDS Worker will launch Microsoft Powershell command. Please check Powershell is correctly installed and configured on your host.
Every command CDS Worker will launch should be in the %PATH% variable.

Do the following steps with the target User whom will run the CDS Worker.

To be able to operate git clone commands, you have to follow next steps :

 * Install Git ` https://git-scm.com/download/win `
 * Setup your system %PATH%, and add `C:\Program Files\Git\mingw32\bin;C:\Program Files\Git\usr\bin`
 * In a Powsershell Prompt, run `PS Set-ExecutionPolicy Unrestricted`
 * You may want to improve your Git Experience on your windows host : have a look to posh-git : https://git-scm.com/book/en/v2/Git-in-Other-Environments-Git-in-Powershell

#### Trigger your pipeline on your Host
If you want to force you pipeline execution on your host, add a "hostname" requirement on you pipeline and set it to the hostname of your windows host.
