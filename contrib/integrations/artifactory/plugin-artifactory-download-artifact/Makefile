.PHONY: clean

VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`
CDSCTL := $(if ${CDSCTL},${CDSCTL},cdsctl)

TARGET_DIR = ./dist
TARGET_NAME = plugin-artifactory-download-artifact

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
endef
export PLUGIN_MANIFEST_BINARY

TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_NAME)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)

GO_BUILD = go build

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

default: build

clean:
	@rm -rf $(TARGET_DIR)

build: $(TARGET_DIR)
	@cp $(TARGET_NAME).yml $(TARGET_DIR)/$(TARGET_NAME).yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo Compiling $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
			FILENAME=$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
			GOOS=$$GOOS GOARCH=$$GOARCH CGO_ENABLED=0 $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$$FILENAME || exit 1; \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_DIR)/plugin-artifactory-download-artifact-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%os%,$$GOOS,g $(TARGET_DIR)/plugin-artifactory-download-artifact-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%arch%,$$GOARCH,g $(TARGET_DIR)/plugin-artifactory-download-artifact-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%filename%,$$FILENAME,g $(TARGET_DIR)/plugin-artifactory-download-artifact-$$GOOS-$$GOARCH.yml; \
		done; \
	done

publish:
	@echo "Updating plugin..."
	$(CDSCTL) admin plugins import $(TARGET_DIR)/$(TARGET_NAME).yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo "Updating plugin binary $(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION"; \
			$(CDSCTL) admin plugins binary-add artifactory-download-artifact-plugin $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH.yml $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
		done; \
	done
