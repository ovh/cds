+++
title = "install"
+++
## worker key install

`worker key install [--env-git] [--env] [--file destination-file] <key-name>`

### Synopsis


Inside a step script you can install a SSH/PGP key generated in CDS in your ssh environment and return the PKEY variable (only for SSH)

So if you want to update your PKEY variable, which is the variable with the path to the SSH private key you just can write PKEY=$(worker key install proj-mykey)` (only for SSH)

You can use the `--env` flag to export the PKEY variable:

```
$ eval $(worker key install --env proj-mykey)
echo $PKEY # variable $PKEY will contains the path of the SSH private key
```

You can use the `--file`  flag to write the private key to a specific path
```
$ worker key install --file .ssh/id_rsa proj-mykey
```

For most advanced usage with git and SSH, you can run `eval $(worker key install --env-git proj-mykey)`.

The `--env-git` flag will display:

```
$ worker key install --env-git proj-mykey
echo "ssh -i /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv -o StrictHostKeyChecking=no \$@" > /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh;
chmod +x /tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh;
export GIT_SSH="/tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv.gitssh.sh";
export PKEY="/tmp/5/0/2569/655/bd925028e70aea34/cds.key.proj-mykey.priv";
```

So that, you can use custom git commands the the previous installed SSH key.



```
worker key install [flags]
```

### Examples

```
worker key install proj-test
```

### Options

```
      --env           display shell command for export $PKEY variable. See documentation.
      --env-git       display shell command for advanced usage with git. See documentation.
      --file string   write key to destination file. See documentation.
  -h, --help          help for install
```

### SEE ALSO

* [worker key](/manual/components/worker/key/)	 - 

