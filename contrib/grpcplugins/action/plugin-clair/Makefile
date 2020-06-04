
TARGET_NAME 			= plugin-clair
TARGET_OS_EXCLUDED  = freebsd

##### ^^^^^^ EDIT ABOVE ^^^^^^ #####

include ../../../../.build/core.mk
include ../../../../.build/go.mk
include ../../../../.build/plugin.mk

.IGNORE: dist/*

build: mk_go_build_plugin ## build action plugin and prepare configuration for publish

publish: mk_plugin_publish ## publish the plugin on CDS. This use your cdsctl default context and commands cdsctl admin plugins import / binary-add.
