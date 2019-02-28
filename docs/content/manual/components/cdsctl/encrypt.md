+++
title = "encrypt"
+++
## cdsctl encrypt

`Encrypt variable into your CDS project`

### Synopsis

To be able to write secret in the CDS yaml files, you have to encrypt data with the project GPG key.

Create a secret variable:


    $ cdsctl encrypt MYPROJECT my-data my-super-secret-value
    my-data: 01234567890987654321

The command returns the value: 01234567890987654321. You can use this value in a configuration file.

Example of use case: Import an environment with a secret.

Create an environment file to import :

    $ cat << EOF > your-environment.yml
    name: your-environment
    values:
    a-readable-variable:
        type: string
        value: registry.ovh.net/engine/http2kafka
    my-data:
        type: password
        value: 01234567890987654321
    EOF


Then, import then environment:

    cdsctl environment import MYPROJECT your-environment.yml

Or push your workflow

	cdsctl workflow push MYPROJECT *.yml


```
cdsctl encrypt [ PROJECT-KEY ] VARIABLE-NAME [SECRET-VALUE] [flags]
```

### Examples

```
cdsctl encrypt MYPROJECT my-data my-super-secret-value
my-data: 01234567890987654321
```

### Options

```
  -h, --help   help for encrypt
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl](/cli/cdsctl/cdsctl/)	 - CDS Command line utility

