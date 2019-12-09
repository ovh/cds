.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot${CDS_VERSION})
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = dist

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
endef
export PLUGIN_MANIFEST_BINARY

GO_BUILD = go build
DIRS = `ls -d */ | cut -f1 -d'/'`

default: build

clean:
	@rm -rf $(TARGET_DIR)

TARGET_DIR =  ./dist
isWindows = $(filter $1,windows)

TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_ENGINE) -X github.com/ovh/cds/sdk.DBMIGRATE=$(DBMIGRATE)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

ALL_PLUGINS := $(if ${TARGET_NAME},${TARGET_NAME}, $(filter-out $(TARGET_DIR), $(shell for plugin in $(DIRS); do echo $(basename "$$plugin"); done)))
TARGET_PLUGINS := $(foreach PLUGIN, $(ALL_PLUGINS), $(TARGET_DIR)/plugin-$(PLUGIN))
TARGET_PLUGINS_DIST := $(foreach TARGET_PLUGIN, $(TARGET_PLUGINS), $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(TARGET_PLUGIN)-$(OS)-$(ARCH)$(if $(call isWindows, $(OS)),.exe))))
TARGET_PLUGINS_MANIFEST_DIST := $(foreach TARGET_PLUGIN, $(TARGET_PLUGINS), $(TARGET_PLUGIN)-$(OS)-$(ARCH).yml)

define get_os_from_binary_file
$(strip $(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n-1]}'))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n]}')))
endef

define get_executor_path_from_binary_file
$(strip $(patsubst dist/%, %, $(patsubst %-, %, $(shell echo $(1) |awk '{n=split($$1,a,"-");for (i = 2; i < n-1; i++) printf a[i] "-"}'))))
endef

build: $(TARGET_PLUGINS_DIST)

$(TARGET_DIR):
	@mkdir $(TARGET_DIR)

$(TARGET_PLUGINS_DIST):
	$(info Compiling $@: $(abspath $@))
	$(MAKE) --no-print-directory gobuild PACKAGE=$(call get_executor_path_from_binary_file,$@) GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@
	$(MAKE) --no-print-directory manifestbuild PACKAGE=$(call get_executor_path_from_binary_file,$@) GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@

gobuild:
	@if go tool dist list | grep "$$GOOS" | grep "$$GOARCH" > /dev/null; then \
		if [ -f $(PACKAGE)/go.mod ]; then \
			cd $(abspath $(PACKAGE)) && GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(abspath $(OUTPUT)); \
		fi; \
	fi

manifestbuild:
	@if go tool dist list | grep "$$GOOS" | grep "$$GOARCH" > /dev/null; then \
		if [ -f $(PACKAGE)/go.mod ]; then \
			echo "$$PLUGIN_MANIFEST_BINARY" > $(TARGET_DIR)/plugin-$$PACKAGE-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%os%,$$GOOS,g $(TARGET_DIR)/plugin-$$PACKAGE-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%arch%,$$GOARCH,g $(TARGET_DIR)/plugin-$$PACKAGE-$$GOOS-$$GOARCH.yml; \
			perl -pi -e s,%filename%,$$FILENAME,g $(TARGET_DIR)/plugin-$$PACKAGE-$$GOOS-$$GOARCH.yml; \
			pluginname=$$PACKAGE; \
			cp $$PACKAGE/$$pluginname.yml $(TARGET_DIR)/plugin-$$PACKAGE.yml; \
		fi; \
	fi

display:
	$(info $(TARGET_PLUGINS_DIST))

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
