.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = ./dist
TARGET_NAME = kubernetes

PLUGIN_MANIFEST=`echo plugin.yml`
export PLUGIN_MANIFEST

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
requirements:
- name: git
  type: binary
  value: git
endef
export PLUGIN_MANIFEST_BINARY

TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_NAME)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

GO_BUILD = go build -v

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

default: build

clean:
	@rm -rf $(TARGET_DIR)

build: $(TARGET_DIR)
	@cp plugin.yml $(TARGET_DIR)/plugin.yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo Compiling $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
			FILENAME=$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$$FILENAME; \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			sed -i "" "s/%os%/$$GOOS/" $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			sed -i "" "s/%arch%/$$GOARCH/" $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			sed -i "" "s/%filename%/$$FILENAME/" $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
		done; \
	done

publish:
	@echo "Updating plugin..."
	cdsctl admin plugins import $(TARGET_DIR)/plugin.yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo "Updating plugin binary $(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION"; \
			cdsctl admin plugins binary-add plugin-kubernetes-deployment $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
		done; \
	done
