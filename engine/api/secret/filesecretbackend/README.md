# File Secret Backend

**Warning: This Secret Backend is not secure: use it only for development or test**

## Files structure

- Create a `.secrets` directory
- Each secret is defined as a file and must have a `.key` extension
- Each file is must be composed of two parts :
  - secret name: the first line of the file always starting with `cds/` prefix
  - secret value: the other lines

## CDS Setup

File Secret Backend is the default mode in CDS. So you don't have to set any option.
You can set the path to your `.secrets` directory (or use a different name) with option `--secret-backend-option "secret_directory=/path/to/my_directory"`. If this option is not set, `.secrets` directory will be loaded.

## Sample usage

### Storing CDS Stash private key

- Create a `my_stash.key` file in our `.secrets` directory. We consider that the repository manager is named `my_stash`.
- Set the following file content

```shell
$ cat my_stash.key
cds/repositoriesmanager-secrets-my_stash-privatekey
-----BEGIN PRIVATE KEY-----
A7qVvdqxevEuUkW4K+jfdkshjfjksdhfhgfhjdhf+0LYmVjPKlJGNXHDGuy5Fw/d
[...]
Lw03eHTNQghS0A==
-----END PRIVATE KEY-----
```

### Storing CDS Github client secret

- Create a `github.key` file in our `.secrets` directory. We consider that the repository manager is named `github`.
- Set the following file content

```shell
$ cat github.key
cds/repositoriesmanager-secrets-github-client-secret
8ed279e27119a85f990e82c7f0b895dd193c6666
```