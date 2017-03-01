
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
		description = "Directory where is Perl Source Code"
		value = "./src"
	}
}

// Steps
steps = [{
	script = <<EOF
#!/bin/bash

set -e

cd {{.testDirectory}}
mkdir -p results
prove -r --timer --formatter=TAP::Formatter::JUnit > results/resultsUnitsTests.xml

EOF
	}, {
		final = true
		artifactUpload = {
				path = "{{.testDirectory}}/results/resultsUnitsTests.xml"
				tag = "{{.cds.version}}"
	  }
	}, {
		final = true
		jUnitReport = "{{.testDirectory}}/results/resultsUnitsTests.xml"
	}]
