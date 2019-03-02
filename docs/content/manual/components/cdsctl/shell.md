+++
title = "shell"
+++
## cdsctl shell

`cdsctl interactive shell`

### Synopsis


CDS Shell Mode. default commands:

- cd: reset current position.
- cd <SOMETHING>: go to an object. Try cd /project/ and tabulation to autocomplete
- find <SOMETHING>: find a project / application / workflow. not case sensitive
- help: display this help
- ls: display current list, quiet format
- ll: display current list
- mode: display current mode. Choose mode with "mode vi" ou "mode emacs"
- open: open CDS WebUI with current context
- version: same as cdsctl version command

Other commands are available depending on your position. Example, run interactively a workflow:


	cd /project/MY_PRJ_KEY/workflow/MY_WF
	run -i

[![asciicast](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk.png)](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk)

[![asciicast](https://asciinema.org/a/H67HlKNS2r0daQaEcuNfZhZZd.png)](https://asciinema.org/a/H67HlKNS2r0daQaEcuNfZhZZd)




```
cdsctl shell [flags]
```

### Options

```
  -h, --help   help for shell
```

### Options inherited from parent commands

```
  -f, --file string   set configuration file
  -k, --insecure      (SSL) This option explicitly allows curl to perform "insecure" SSL connections and transfers.
  -w, --no-warnings   do not display warnings
  -v, --verbose       verbose output
```

### SEE ALSO

* [cdsctl](/manual/components/cdsctl/cdsctl/)	 - CDS Command line utility

