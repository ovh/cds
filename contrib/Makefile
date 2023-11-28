.PHONY: clean build package copy

TARGET_DIST := $(if ${TARGET_DIST},${TARGET_DIST},./dist)

clean:
	$(MAKE) clean -C grpcplugins/action
	$(MAKE) clean -C integrations
	rm -rf $(TARGET_DIST)

ALL_DISTS := $(shell find . -type d -name "dist")
ALL_FILES := $(foreach d, $(ALL_DISTS), $(wildcard $(d)/*))

build:
	$(MAKE) build -C grpcplugins/action OS="$(OS)" ARCH="$(ARCH)"
	$(MAKE) build -C integrations OS="$(OS)" ARCH="$(ARCH)"

$(TARGET_DIST):
	@mkdir -p $(TARGET_DIST)

ALL_TARGETS := $(foreach f, $(ALL_FILES), $(TARGET_DIST)/$(notdir $(f)))

$(ALL_TARGETS): $(TARGET_DIST)
	$(info moving $(filter %/$(notdir $@), $(ALL_FILES)) to $@)
	@mv -f $(filter %/$(notdir $@), $(ALL_FILES)) $@

package: $(ALL_TARGETS) $(TARGET_DIST)

ALL_PLUGINS = $(filter-out null,$(foreach f,$(wildcard $(TARGET_DIST)/*.yml), $(shell cat $(f) | yq .name)))
ALL_ARCHIVES = $(foreach f,$(ALL_PLUGINS),$(strip $(TARGET_DIST)/$(f))-all.tar.gz)

$(ALL_ARCHIVES): ARCHIVE = $(notdir $@)
$(ALL_ARCHIVES): PATTERN = $(subst -all.tar.gz,*[!.tar.gz],$(ARCHIVE))
$(ALL_ARCHIVES):
	$(info Preparing archive $(ARCHIVE) from $(PATTERN))
	cd $(TARGET_DIST); tar czf $(ARCHIVE) $(PATTERN)

archive: $(ALL_ARCHIVES)

clean_archive:
	rm -rf $(TARGET_DIST)/*.tar.gz
