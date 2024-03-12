SHELL := /bin/bash
BINARIES_CONF                   = $(addprefix $(TARGET_DIST)/, $(addsuffix -$(OS)-$(ARCH).yml, $(notdir $(TARGET_NAME))))
PLUGIN_CONF                     = $(addprefix $(TARGET_DIST)/, $(addsuffix .yml, $(notdir $(TARGET_NAME))))
CROSS_COMPILED_PLUGIN_CONF      = $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(if $(IS_OS_ARCH_VALID), $(BINARIES_CONF))))

.PHONY: build clean test publish package

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

define get_executor_path_from_binary_file
$(strip $(patsubst dist/%, %, $(patsubst %-, %, $(shell echo $(1) |awk '{n=split($$1,a,"-");for (i = 2; i < n-1; i++) printf a[i] "-"}'))))
endef

define PLUGIN_CDS_NAME
$(if $(PLUGIN_NAME),$(PLUGIN_NAME),$(TARGET_NAME))
endef
export PLUGIN_CDS_NAME

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
	if test "$(OS)" = "windows" ; then EXTENSION=".exe"; fi; \
	FILENAME=$(TARGET_NAME)-$$OS-$$ARCH$$EXTENSION; \
	perl -pi -e s,%filename%,$$FILENAME,g $@

$(PLUGIN_CONF):
	$(info *** prepare conf $@)
	@mkdir -p $(TARGET_DIST); \
	cp $(TARGET_NAME).yml $@;

OPT_LD_FLAGS = -s -w
mk_go_build_plugin: $(CROSS_COMPILED_PLUGIN_CONF) $(PLUGIN_CONF) $(CROSS_COMPILED_BINARIES)

mk_plugin_publish:
	@echo "Updating plugin $(TARGET_NAME) for $(TARGET_OS), $(TARGET_ARCH)..."
	cdsctl admin plugins import $(TARGET_DIST)/$(TARGET_NAME).yml
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			for V in $(OSARCHVALID); do \
				if test "$$GOOS/$$GOARCH" = "$$V"; then \
					if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
					echo "Updating plugin binary $(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION"; \
					cdsctl admin plugins binary-add $(TARGET_NAME) $(TARGET_DIST)/$(TARGET_NAME)-$$GOOS-$$GOARCH.yml $(TARGET_DIST)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
				fi; \
			done; \
		done; \
	done

mk_plugin_package:
	tar czf $(TARGET_DIST)/cds-$(TARGET_NAME)-all.tar.gz $(TARGET_DIST)/$(TARGET_NAME)*

mk_v2_plugin_publish:
	@echo "Updating plugin $(PLUGIN_CDS_NAME) for $(TARGET_OS) $(TARGET_ARCH)..."
	cdsctl experimental plugin import $(TARGET_DIST)/$(TARGET_NAME).yml --force
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			for V in $(OSARCHVALID); do \
				if test "$$GOOS/$$GOARCH" = "$$V"; then \
					if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
					echo "Updating plugin binary $(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION"; \
					cdsctl admin plugins binary-add $$PLUGIN_CDS_NAME $(TARGET_DIST)/$(TARGET_NAME)-$$GOOS-$$GOARCH.yml $(TARGET_DIST)/$(TARGET_NAME)-$$GOOS-$$GOARCH$$EXTENSION; \
				fi; \
			done; \
		done; \
	done
