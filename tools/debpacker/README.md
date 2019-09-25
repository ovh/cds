# DebPacker

Creates a Debian package and a systemd configuration

## prerequisite

* You must have a sane instance of `go`
* You must be in a debian environment (needs `dpkg`)

## Usage

The program needs a configuration file to work. This configuration file describes how to build the package

### package-name

**Type:** *string*

Name of the debian package

### architecture

**Type:** *string*

Architecture of the system. For instance `armhf`, `all`

### binary-file

**Type:** *string*

local path to the binary file to execute

### command

**Type:** *string*

Command

### configuration-files

**Type:** *string[]*

List of configuration files

### copy-files

**Type:** *string[]*

List of files to copy. Syntax: `path[:dest]`

For instance:

`./foo/*.txt` copy all files that match the pattern
`./foo/*.txt:data` copy all files that match the pattern in `/var/lib/{package name}/data`
`./foo/*.conf:/etc/foo` copy all files that match the pattern in `/etc/foo`

### mkdirs

**Type:** *string[]*

List of folders to create

For instance:

`foo/mickey` creates the folder `/var/lib/{package name}/foo/mickey`
`/etc/foo` creates the folder `/etc/foo`

### version

**Type:** *string*

Package version

### description

**Type:** *string*

Package description

### maintainer

**Type:** *string*

Email of the maintener

### dependencies

**Type:** *string[]*

List of dependancies

### systemd-configuration

Configure service

#### after

**Type:** *string*

After

#### user

**Type:** *string*

User that launch the binary

#### args

**Type:** *string[]*

Arguments to pass to the binary

#### stop-command

**Type:** *string*

Stop command

#### restart

**Type:** *string*

Restart policy. For instance `always`

#### wanted-by

**Type:** *string*

Wanted by

#### environments

**Type:** *dictionary*

Environment variables

#### working-directory

**Type:** *string*

Directory in which the binary is launched

#### post-install-cmd

**Type:** *string*

Post install command

#### auto-start

**Type:** *boolean*

If true, start the service during package installation

## Examples

### Executable

```yaml
package-name: "my-executable"
architecture: "all"
binary-file: "./my-exec"
copy-files:
  - "./data/*.txt:data"
version: "0.0.1"
description: "Executable example"
maintainer: "mickey@mouse.com"
systemd-configuration:
  user: "root"
  after: network.target
  stop-command: /bin/kill $MAINPID
  restart: always
  wanted-by: multi-user.target
  working-directory: "/var/lib/foo"
```

### html client

```yaml
package-name: "my-website"
architecture: "all"
binary-file: "./my-exec"
copy-files:
  - "./data/*:/var/www"
version: "0.0.1"
description: "Executable example"
maintainer: "mickey@mouse.com"
systemd-configuration:
  user: "root"
  after: network.target
  wanted-by: multi-user.target
  post-install-cmd: "systemctl restart nginx"

```
