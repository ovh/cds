+++
title = "Worker Setup"
weight = 1

[menu.main]
parent = "tutorials"
identifier = "tutorials-worker-setup"

+++


### Introduction

#### Why would you need to setup your own worker ?

There is several cases where one would need to setup his own worker:

 * Perform incremental build
 * Build on a specific architecture
 * Perform integration tests in a specific network

### How does this work ?

Workers authenticate on CDS with a [token]({{< relref "advanced.worker.token.md" >}}) and have the same permissions as the user who generated it.

Bottom line, if you can access the application, your worker will too.

### Linux Setup

#### Download the binary

Simple enough, run

```bash
$ wget -nv https://your-cds-api/download/worker/`uname -m` -O worker
```

or download from https://github.com/ovh/cds/releases

#### Startup the worker

```bash
$ worker --help
CDS Worker

Usage:
  worker [flags]
  worker [command]

Available Commands:
  export      worker export <varname> <value>
  upload      worker upload --tag=<tag> <path>
  version     Print the version number
  register    worker register

Flags:
      --api string                   URL of CDS API
      --basedir string               Worker working directory
      --booked-job-id int            Booked job id
      --graylog-extra-key string     Ex: --graylog-extra-key=xxxx-yyyy
      --graylog-extra-value string   Ex: --graylog-extra-value=xxxx-yyyy
      --graylog-host string          Ex: --graylog-host=xxxx-yyyy
      --graylog-port string          Ex: --graylog-port=12202
      --graylog-protocol string      Ex: --graylog-protocol=xxxx-yyyy
      --grpc-api string              CDS GRPC tcp address
      --grpc-insecure                Disable GRPC TLS encryption
      --hatchery int                 Hatchery spawing worker
      --token string                 CDS Token
      --log-level string             Log Level : debug, info, notice, warning, critical (default "notice")
      --model int                    Model of worker
      --name string                  Name of worker
      --single-use                   Exit after executing an action
      --ttl int                      Worker time to live (minutes) (default 30)

Use "worker [command] --help" for more information about a command.
```

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
