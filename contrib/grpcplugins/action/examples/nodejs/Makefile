.PHONY: clean

VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)

TARGET_DIR = dist
TARGET_NAME = nodejs

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: node
entrypoints:
- index.js
requirements:
- name: node
  type: binary
  value: node
endef
export PLUGIN_MANIFEST_BINARY

TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)
TARGET_TAR = plugin-$(TARGET_NAME).tar.gz

build:
	@mkdir -p $(TARGET_DIR)
	@cp $(TARGET_NAME).yml $(TARGET_DIR)/plugin.yml
	@npm install
	@npm link ./../../../../../sdk/grpcplugin/actionplugin/nodejs
	@tar -cHhzvf $(TARGET_DIR)/$(TARGET_TAR) node_modules *.js *.json *.yml
	@for OS in $(TARGET_OS); do \
		for ARCH in $(TARGET_ARCH); do \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_DIR)/plugin-$$OS-$$ARCH.yml; \
			perl -pi -e s,%os%,$$OS,g $(TARGET_DIR)/plugin-$$OS-$$ARCH.yml; \
			perl -pi -e s,%arch%,$$ARCH,g $(TARGET_DIR)/plugin-$$OS-$$ARCH.yml; \
		done; \
	done;

publish:
	@echo "Updating plugin $(TARGET_NAME)..."
	cdsctl admin plugins import $(TARGET_DIR)/plugin.yml
	@for OS in $(TARGET_OS); do \
		for ARCH in $(TARGET_ARCH); do \
			echo "Updating plugin binary $(TARGET_NAME)-$$OS-$$ARCH"; \
			cdsctl admin plugins binary-add plugin-$(TARGET_NAME) $(TARGET_DIR)/plugin-$$OS-$$ARCH.yml $(TARGET_DIR)/$(TARGET_TAR); \
		done; \
	done
