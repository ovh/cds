+++
title = "cdsctl"
+++
## cdsctl

CDS Command line utility

### Synopsis



## Download

You'll find last release of `cdsctl` on [Github Releases](https://github.com/ovh/cds/releases/latest).


## Authentication

Per default, the command line `cdsctl` uses your keychain on your os:

* OSX: Keychain Access
* Linux System: Secret-tool (libsecret)
* Windows: Windows Credentials service

You can bypass keychain tools by using environment variables:

	CDS_API_URL="https://instance.cds.api" CDS_USER="username" CDS_TOKEN="yourtoken" cdsctl [command]


Want to debug something? You can use `CDS_VERBOSE` environment variable.

	CDS_VERBOSE=true cdsctl [command]


If you're using a self-signed certificate on CDS API, you probably want to use `CDS_INSECURE` variable.

	CDS_INSECURE=true cdsctl [command]



```
cdsctl [flags]
```

### Options

```
  -f, --file string   set configuration file
  -h, --help          help for cdsctl
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl action](/manual/components/cdsctl/action/)	 - `Manage CDS action`
* [cdsctl admin](/manual/components/cdsctl/admin/)	 - `Manage CDS (admin only)`
* [cdsctl application](/manual/components/cdsctl/application/)	 - `Manage CDS application`
* [cdsctl encrypt](/manual/components/cdsctl/encrypt/)	 - `Encrypt variable into your CDS project`
* [cdsctl environment](/manual/components/cdsctl/environment/)	 - `Manage CDS environment`
* [cdsctl group](/manual/components/cdsctl/group/)	 - `Manage CDS group`
* [cdsctl health](/manual/components/cdsctl/health/)	 - `Check CDS health`
* [cdsctl login](/manual/components/cdsctl/login/)	 - `Login to CDS`
* [cdsctl monitoring](/manual/components/cdsctl/monitoring/)	 - `CDS monitoring`
* [cdsctl pipeline](/manual/components/cdsctl/pipeline/)	 - `Manage CDS pipeline`
* [cdsctl project](/manual/components/cdsctl/project/)	 - `Manage CDS project`
* [cdsctl shell](/manual/components/cdsctl/shell/)	 - `cdsctl interactive shell`
* [cdsctl signup](/manual/components/cdsctl/signup/)	 - `Signup on CDS`
* [cdsctl template](/manual/components/cdsctl/template/)	 - `Manage CDS workflow template`
* [cdsctl token](/manual/components/cdsctl/token/)	 - `Manage CDS group token`
* [cdsctl update](/manual/components/cdsctl/update/)	 - `Update cdsctl from CDS API or from CDS Release`
* [cdsctl user](/manual/components/cdsctl/user/)	 - `Manage CDS user`
* [cdsctl version](/manual/components/cdsctl/version/)	 - `show cdsctl version`
* [cdsctl worker](/manual/components/cdsctl/worker/)	 - `Manage CDS worker`
* [cdsctl workflow](/manual/components/cdsctl/workflow/)	 - `Manage CDS workflow`

