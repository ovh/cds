+++
title = "Setup your worker"
weight = 1

+++


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
$ worker --api http://localhost:8081 --token 5459c00e70f31f4bd2c9762660983fff641a5e6b8cffc322a68586a95ed65c7f
INFO[0000] running worker CDS_SINGLE_USE:false...
INFO[0000] Current binary: /home/cds/worker
2018-01-17 22:27:40 [INFO] CDS Worker starting
2018-01-17 22:27:40 [INFO] version: 0.25.1-snapshot+1693.cds
2018-01-17 22:27:40 [INFO] hostname: localhost.local
2018-01-17 22:27:40 [INFO] auto-update: false
2018-01-17 22:27:40 [INFO] single-use: false
2018-01-17 22:27:40 [INFO] Export variable HTTP server: 127.0.0.1:64781
2018-01-17 22:27:40 [INFO] Registering on CDS engine Version:0.25.1-snapshot+1693.cds
2018-01-17 22:27:40 [INFO] Registering localhost.local on http://localhost:8081
2018-01-17 22:27:40 [INFO] localhost.local Registered on http://localhost:8081
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

