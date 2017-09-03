
name = "CDS_PerlTest"
description = "Test with prove on perl source code"

// Requirements
requirements = {
  "perl" = {
		type = "binary"
		value = "perl"
	}
	"bash" = {
		type = "binary"
		value = "bash"
	}
	"prove" = {
		type = "binary"
		value = "prove"
	}
}

// Parameters
parameters = {
	 "testDirectory" = {
		type = "string"
		description = "Directory in which prove will be launched"
		value = "./src"
	}
	 "proveOptions" = {
		type = "string"
		description = "Options passed to prove"
		value = "-r --timer"
	}
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash

set -e

cd {{.testDirectory}}
mkdir -p results
prove --formatter=TAP::Formatter::JUnit {{.proveOptions}} > results/resultsUnitsTests.xml

EOF
	}, {
		always_executed = true
		artifactUpload = {
				path = "{{.testDirectory}}/results/resultsUnitsTests.xml"
				tag = "{{.cds.version}}"
	  }
	}, {
		always_executed = true
		jUnitReport = "{{.testDirectory}}/results/resultsUnitsTests.xml"
	}]
