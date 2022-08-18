.PHONY: clean doc modclean mod goinstall build dist deb package_contrib

TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)
VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
GIT_DESCRIBE := $(shell git describe --tags)
GIT_VERSION := $(if ${GIT_DESCRIBE},${GIT_DESCRIBE},0.0.0-0-snapshot)
SHA512 := $(if ifeq ${UNAME} "Darwin",shasum -a 512,sha512sum)

TARGET_ENGINE = engine
TARGET_WORKER = worker
TARGET_CDSCTL = cdsctl

doc:
ifndef GEN_PATH
	$(error GEN_PATH is undefined)
endif
	# GEN_PATH=./docs/content/docs/components
	$(TARGET_CDSCTL) doc $(GEN_PATH)
	$(TARGET_WORKER) doc $(GEN_PATH)
	$(TARGET_ENGINE) doc $(GEN_PATH) ./
	cd docs && ./build.sh

modclean:
	@echo "cleaning modcache... " && GO111MODULE=off go clean -modcache || true


ENGINE_DIST = $(wildcard engine/dist/*)
WORKER_DIST = $(wildcard engine/worker/dist/*)
CLI_DIST = $(wildcard cli/cdsctl/dist/*)
CONTRIB_DIST = $(wildcard contrib/dist/*)
UI_DIST = ui/ui.tar.gz
FILES = dist/FILES

TARGET_DIR := dist/
ALL_DIST = $(ENGINE_DIST)
ALL_DIST := $(ALL_DIST) $(WORKER_DIST)
ALL_DIST := $(ALL_DIST) $(CLI_DIST)
ALL_DIST := $(ALL_DIST) $(UI_DIST)
ALL_DIST := $(ALL_DIST) $(CONTRIB_DIST)
ALL_TARGETS := $(foreach DIST,$(ALL_DIST),$(addprefix $(TARGET_DIR),$(notdir $(DIST))))

CONTRIB_ALL_FILES := $(wildcard contrib/dist/*)

build:
	$(info Building CDS Components for $(TARGET_OS) - $(TARGET_ARCH))
	$(MAKE) build_ui -j1
	$(MAKE) build_engine -j4
	$(MAKE) build_worker -j4
	$(MAKE) build_cli -j4
	$(MAKE) build_contrib -j4
	$(MAKE) package_contrib

package_contrib: $(TARGET_DIR)
	$(info copying $(CONTRIB_ALL_FILES))
	@for f in ${CONTRIB_ALL_FILES} ; do echo "copying $$f to ${TARGET_DIR}`basename $$f`"; cp $$f ${TARGET_DIR}`basename $$f`; done

build_ui:
	$(MAKE) build -C ui

build_engine:
	$(MAKE) build -C engine OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

build_worker:
	$(MAKE) build -C engine/worker OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

build_cli:
	$(MAKE) build -C cli/cdsctl OS="$(foreach OS,${TARGET_OS},${OS}/%)" ARCH="$(foreach ARCH,${TARGET_ARCH},%/${ARCH})"

build_contrib:
	 $(MAKE) build -C contrib OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

define get_dist_from_target
$(filter %/$(notdir $(1)), $(ALL_DIST))
endef

$(TARGET_DIR):
	@mkdir -p $(TARGET_DIR)

$(ALL_TARGETS): $(TARGET_DIR)
	$(info copying $(call get_dist_from_target, $@) to $@)
	@cp -f $(call get_dist_from_target, $@) $@

dist: $(ALL_TARGETS)
	$(info sha512 = ${SHA512})
	rm -f $(FILES)
	cd dist/ && for i in `ls -p | grep -v /|grep -v FILES`; do echo "$$i;`${SHA512} $$i|cut -d ' ' -f1`" >> FILES; done;

clean:
	@rm -rf target
	@rm -rf dist
	$(MAKE) clean -C engine
	$(MAKE) clean -C engine/worker
	$(MAKE) clean -C cli/cdsctl
	$(MAKE) clean -C ui
	$(MAKE) clean -C contrib

deb: dist target/cds-engine.deb

$(TARGET_DIR)/config.toml.sample:
	$(TARGET_DIR)/cds-engine-linux-amd64 config new > $(TARGET_DIR)/config.toml.sample

target/cds-engine.deb: $(TARGET_DIR)/config.toml.sample
	debpacker make --workdir dist --config .debpacker.yml --var version=$(GIT_VERSION)

docker:
	docker build --tag ovhcom/cds-engine:$(VERSION) .

tar: target/cds-engine.tar.gz

target/cds-engine.tar.gz: $(TARGET_DIR)/config.toml.sample $(TARGET_DIR)/tmpl-config
	mkdir -p target
	tar -czvf target/cds-engine.tar.gz -C $(TARGET_DIR) .

PLUGINS := `ls -d contrib/grpcplugins/action/plugin-*`

tidy:
	@echo "Running tidy on cds main project"
	@go mod tidy
	@for P in $(PLUGINS); do \
		echo "Running tidy on $$P"; \
		(cd $$P && go mod tidy); \
	done;
	@echo "Running tidy on tests/fixtures/04SCWorkflowRunSimplePlugin"
	@(cd tests/fixtures/04SCWorkflowRunSimplePlugin && go mod tidy)
	@(cd sdk/interpolate && go mod tidy)
	@(cd tools/smtpmock && go mod tidy)
