SHELL := /bin/bash
BINARIES_CONF                   = $(addprefix $(TARGET_DIST)/, $(addsuffix -$(OS)-$(ARCH).yml, $(notdir $(TARGET_NAME))))
PLUGIN_CONF                     = $(addprefix $(TARGET_DIST)/, $(addsuffix .yml, $(notdir $(TARGET_NAME))))
CROSS_COMPILED_PLUGIN_CONF      = $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(if $(IS_OS_ARCH_VALID), $(BINARIES_CONF))))

.PHONY: build clean test package

define PLUGIN_MANIFEST_BINARY
os: %os%
arch: %arch%
cmd: ./%filename%
endef
export PLUGIN_MANIFEST_BINARY

define get_os_from_binary_file
$(strip $(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n-1]}'))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n]}')))
endef

define get_arch_from_conf_file
$(strip $(patsubst %.yml, %,$(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n]}')))
endef

## Prepare yml file for each os-arch
$(CROSS_COMPILED_PLUGIN_CONF): $(GOFILES)
	$(info *** prepare conf $@, TARGET_NAME=$(TARGET_NAME))
	@mkdir -p $(TARGET_DIST); \
	echo "$$PLUGIN_MANIFEST_BINARY" > $@; \
	OS=$(call get_os_from_binary_file,$@); \
	ARCH=$(call get_arch_from_conf_file,$@); \
	perl -pi -e s,%os%,$$OS,g $@ ; \
	perl -pi -e s,%arch%,$$ARCH,g $@ ; \
	EXTENSION=""; \
	if test "$$OS" = "windows" ; then EXTENSION=".exe"; fi; \
	FILENAME=$(TARGET_NAME)-$$OS-$$ARCH$$EXTENSION; \
	perl -pi -e s,%filename%,$$FILENAME,g $@

$(PLUGIN_CONF):
	$(info *** prepare conf $@)
	@mkdir -p $(TARGET_DIST); \
	cp $(TARGET_NAME).yml $@;

OPT_LD_FLAGS = -s -w
mk_go_build_plugin: $(CROSS_COMPILED_PLUGIN_CONF) $(PLUGIN_CONF) $(CROSS_COMPILED_BINARIES)

mk_plugin_package:
	tar czf $(TARGET_DIST)/cds-$(TARGET_NAME)-all.tar.gz $(TARGET_DIST)/$(TARGET_NAME)*
