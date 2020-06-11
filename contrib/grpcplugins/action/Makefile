##### =====> Commands <===== #####

include ../../../.build/core.mk
include ../../../.build/go.mk

DIRS 			:= `ls -d plugin-* | cut -f1 -d'/'`
ALL_PLUGINS 	:= $(if ${TARGET_NAME},${TARGET_NAME}, $(filter-out $(TARGET_DIST), $(shell for plugin in $(DIRS); do echo $(basename "$$plugin"); done)))
TARGET_PLUGINS 	:= $(foreach PLUGIN, $(ALL_PLUGINS), dist/$(PLUGIN).yml)
TARGET_PLUGINS_DIST := $(foreach TARGET_PLUGIN, $(TARGET_PLUGINS), $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(TARGET_PLUGIN)-$(OS)-$(ARCH)$(if $(call isWindows, $(OS)),.exe))))
TARGET_DIST 		:= ./dist

define get_plugin_name_from_conf
$(strip $(patsubst dist/%.yml, %,$(shell echo $(1))))
endef

$(TARGET_PLUGINS):
	@mkdir -p dist
	$(info building $(call get_plugin_name_from_conf,$@))
	@P=$(call get_plugin_name_from_conf,$@); \
	$(MAKE) build -C $$P

build: $(TARGET_PLUGINS)

.PHONY: dist test package publish clean

package:
	@mkdir -p dist
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) package -C $$P; \
	done;
	@for PLU in $(ALL_PLUGINS); do \
		mv $$PLU/dist/*.tar.gz dist/; \
	done;

test:
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) test -C $$P; \
	done;

publish: 
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) publish -C $$P; \
	done;

clean: mk_go_clean ## clean binary and tests results
	@rm -rf $(TARGET_DIST)
	@for P in $(ALL_PLUGINS); do \
		$(MAKE) clean -C $$P; \
	done;
