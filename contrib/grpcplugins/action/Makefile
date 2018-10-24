.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = dist
TARGET_NAME = $(filter-out $@ examples,$(MAKECMDGOALS))

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
endef
export PLUGIN_MANIFEST_BINARY

TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_NAME)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

GO_BUILD = GO111MODULE=on go build -mod=vendor -v
DIRS = `ls -d */ | cut -f1 -d'/'`

default: build-all

clean:
	@rm -rf */$(TARGET_DIR)

build-all:
	@for plugin in $(DIRS); do \
		pluginname=$(basename "$$plugin"); \
		mkdir -p $$plugin/$(TARGET_DIR); \
		cp $$plugin/$$pluginname.yml $$plugin/$(TARGET_DIR)/plugin.yml; \
		for GOOS in $(TARGET_OS); do \
			for GOARCH in $(TARGET_ARCH); do \
				EXTENSION=""; \
				if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
				echo Compiling $$plugin/$(TARGET_DIR)/$$pluginname-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
				FILENAME=$$pluginname-$$GOOS-$$GOARCH$$EXTENSION; \
				GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $$plugin/$(TARGET_DIR)/$$FILENAME ./$$pluginname; \
				echo "$$PLUGIN_MANIFEST_BINARY" > $$plugin/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
				perl -pi -e s,%os%,$$GOOS,g $$plugin/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
				perl -pi -e s,%arch%,$$GOARCH,g $$plugin/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
				perl -pi -e s,%filename%,$$FILENAME,g $$plugin/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			done; \
		done; \
	done

publish-all:
	@echo "Updating plugins..."
	@for plugin in $(DIRS); do \
		pluginname=$(basename "$$plugin"); \
		cdsctl admin plugins import $$plugin/$(TARGET_DIR)/plugin.yml; \
		for GOOS in $(TARGET_OS); do \
			for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo "Updating plugin binary $$pluginname-$$GOOS-$$GOARCH$$EXTENSION"; \
			cdsctl admin plugins binary-add plugin-$$pluginname $$plugin/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml $$plugin/$(TARGET_DIR)/$$pluginname-$$GOOS-$$GOARCH$$EXTENSION; \
			done; \
		done; \
	done;

build:
	@mkdir -p $(TARGET_NAME)/$(TARGET_DIR)
	@cp $(TARGET_NAME)/$(TARGET_NAME).yml $(TARGET_NAME)/$(TARGET_DIR)/plugin.yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo Compiling $(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
			FILENAME=$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_NAME)/$(TARGET_DIR)/$$FILENAME ./$(TARGET_NAME); \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_NAME)/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%os%,$$GOOS,g $(TARGET_NAME)/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%arch%,$$GOARCH,g $(TARGET_NAME)/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%filename%,$$FILENAME,g $(TARGET_NAME)/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml; \
		done; \
	done

publish:
	@echo "Updating plugin $(TARGET_NAME)..."
	cdsctl admin plugins import $(TARGET_NAME)/$(TARGET_DIR)/plugin.yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo "Updating plugin binary $(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION"; \
			cdsctl admin plugins binary-add plugin-$(TARGET_NAME) $(TARGET_NAME)/$(TARGET_DIR)/plugin-$$GOOS-$$GOARCH.yml $(TARGET_NAME)/$(TARGET_DIR)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
		done; \
	done
