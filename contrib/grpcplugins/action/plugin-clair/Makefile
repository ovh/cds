
TARGET_NAME 		= plugin-clair
TARGET_OS_EXCLUDED  = freebsd openbsd windows

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

include ../../../../.build/core.mk
include ../../../../.build/go.mk
include ../../../../.build/plugin.mk

build: mk_go_build_plugin ## build action plugin and prepare configuration for publish

clean: mk_go_clean ## clean binary and tests results

test: ## run unit tests
	@echo "No test for $(TARGET_NAME)"

publish: mk_plugin_publish ## publish the plugin on CDS. This use your cdsctl default context and commands cdsctl admin plugins import / binary-add.

package: mk_plugin_package ## prepare the tar.gz file, with all binaries / conf files
