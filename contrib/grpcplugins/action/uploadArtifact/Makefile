PLUGIN_NAME		= uploadArtifact
TARGET_NAME		= uploadArtifact

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

include ../../../../.build/core.mk
include ../../../../.build/go.mk
include ../../../../.build/plugin.mk

build: mk_go_build_plugin ## build action plugin and prepare configuration for publish

clean: mk_go_clean ## clean binary and tests results

test: mk_go_test ## run unit tests
package: mk_plugin_package ## prepare the tar.gz file, with all binaries / conf files
