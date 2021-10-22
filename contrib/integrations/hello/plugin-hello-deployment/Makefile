.PHONY: clean

VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = ./dist
TARGET_NAME = hello

PLUGIN_MANIFEST=`echo $TARGET_NAME.yml`
export PLUGIN_MANIFEST

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
endef
export PLUGIN_MANIFEST_BINARY

TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_NAME)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd openbsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)

GO_BUILD = go build

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
			FILENAME=deployment-plugin-$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
			echo Compiling $(TARGET_DIR)/$$FILENAME $(VERSION); \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$$FILENAME || exit 1; \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_DIR)/plugin-kubernetes-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%os%,$$GOOS,g $(TARGET_DIR)/plugin-kubernetes-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%arch%,$$GOARCH,g $(TARGET_DIR)/plugin-kubernetes-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%filename%,$$FILENAME,g $(TARGET_DIR)/plugin-kubernetes-$$GOOS-$$GOARCH.yml; \
		done; \
	done

publish:
	@echo "Updating plugin..."
	cdsctl admin plugins import $(TARGET_DIR)/plugin.yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			FILENAME=deployment-plugin-$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
      if [ -f "$(TARGET_DIR)/$$FILENAME" ]; then \
        echo "Updating plugin binary $$FILENAME"; \
        PLUGIN_YML=$(TARGET_DIR)/plugin.yml; \
        NAME_PLUGIN=`grep 'name:' $$PLUGIN_YML|cut -d ':' -f2;`; \
        echo "Updating plugin $$NAME_PLUGIN with binary $$FILENAME"; \
        cdsctl admin plugins binary-add $$NAME_PLUGIN $(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml $(TARGET_DIR)/$$FILENAME; \
      else \
        echo "$$FILENAME is missing"; \
      fi \
		done; \
	done
