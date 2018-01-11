+++
title = "cds-python-pylint"

+++

Run pylint.
By default, each pylint recommendation will be displayed as a Junit test.
The short errors are fully contained inside the test title,
an arrow (->) will be present if a part of the recommendation is displayed in
the test body.

## Parameters

* **extra_options**: Extra options to pass during pylint invocation.
* **ignore**: List of ignored files / directory (base name, not path), separated
by a ;
* **module_path**: List of modules paths (absolute or relative) to launch pylint into, separated by a ;.
If empty, will launch pylint inside the working directory
* **pylintrc**: Path of the pylintrc file, or its content.
If your pylintrc file is not used, try to use an absolute path using the variable {{.cds.workspace}} that points to the container default working directory
* **raw_output**: Skip the xunit + Junit step and output a raw pylint result file.
* **raw_output_file**: File to output the raw result if raw output is checked. If empty, will only log the results.
* **xml_output_file**: File to output the result xunit xml, it should not be empty.


## Requirements

* **python3.5**: type: binary Value: python3.5
* **virtualenv**: type: binary Value: virtualenv


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-python-pylint.hcl)


