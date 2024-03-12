.PHONY: clean

VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = ./dist
TARGET_BINARY = cdsctl
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_BINARY)"

split = $(word $2,$(subst /, ,$1))
TARGET_OS := $(if ${OS},${OS}, linux/% darwin/% windows/% freebsd/% openbsd/%)
TARGET_ARCH := $(if ${ARCH}, ${ARCH}, %/amd64 %/arm %/386 %/arm64)
TARGET_DISTS := $(filter $(TARGET_OS), $(filter $(TARGET_ARCH), $(shell go tool dist list)))
isWindows = $(filter $1,windows)
TARGET_BINARIES := $(foreach DIST, $(filter-out darwin/386,$(TARGET_DISTS)), $(TARGET_DIR)/cdsctl-$(call split,$(DIST),1)-$(call split,$(DIST),2)$(if $(call isWindows, $(call split,$(DIST),1)),.exe))
TARGET_BINARIES_VARIANT := $(addsuffix -nokeychain, $(filter %linux-amd64 %darwin-amd64 %darwin-arm64,$(TARGET_BINARIES)))

GO_BUILD = go build
GO_LINT = ${GOPATH}/bin/gometalinter
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GO2XUNIT = ${GOPATH}/bin/go2xunit

PKGS_COMMA_SEP = go list -f '{{ join .Deps "\n" }}{{"\n"}}{{.ImportPath}}' . | grep github.com/ovh/cds | grep -v vendor | tr '\n' ',' | sed 's/,$$//'

define get_os_from_binary_file
$(strip $(shell echo $(1) | cut -d'-' -f 2))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | cut -d'-' -f 3)))
endef


TEST_PKGS = $(shell go list ./... | grep -v vendor)
TEST_CMD = go test -tags skipkeychaintests -v -timeout 180s -coverprofile=profile.coverprofile

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

$(GO_LINT):
	go get -u github.com/alecthomas/gometalinter

$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

$(GO_GO2XUNIT):
	go get -u github.com/tebeka/go2xunit

default: build

clean:
	@rm -rf $(TARGET_DIR)
	@for TST in `find . -name "tests.log"`; do \
		rm $$TST; \
	done;
	@for profile in `find . -name "*.coverprofile"`; do \
		rm $$profile; \
	done;

$(TARGET_BINARIES):
	$(info *** building binary $@)
	@$(MAKE) --no-print-directory gobuild TAGS='' CGO_ENABLED=1 GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@

$(TARGET_BINARIES_VARIANT):
	$(info *** building binary variant $@)
	@$(MAKE) --no-print-directory gobuild TAGS="--tags='nokeychain'" CGO_ENABLED=0 GOOS=$(call get_os_from_binary_file,$(subst -nokeychain,,$@)) GOARCH=$(call get_arch_from_binary_file,$(subst -nokeychain,,$@)) OUTPUT=$@

gobuild:
	$(info ... OS:$(GOOS) ARCH:$(GOARCH) -> $(OUTPUT))
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) $(GO_BUILD) $(TAGS) $(TARGET_LDFLAGS) -o $(OUTPUT)

build: $(TARGET_DIR) $(TARGET_BINARIES_VARIANT) $(TARGET_BINARIES)

build_test_cdsctl:
	TEMP=`$(PKGS_COMMA_SEP)`; \
	CGO_ENABLED=1 go test -coverpkg $$TEMP -c . -o $(TARGET_DIR)/$(TARGET_BINARY).test -tags testbincover,nokeychain;

test: clean
	@for PKG in $(TEST_PKGS); do \
		echo "Running tests in package $$PKG"; \
		cd ${GOPATH}/src/$$PKG; \
		$(TEST_CMD) > ${GOPATH}/src/$$PKG/tests.log; \
	done;
	$(MAKE) test-coverage
	$(MAKE) test-xunit-report

test-coverage: $(GO_COV_MERGE) $(TARGET_DIR)
	@$(GO_COV_MERGE) ./*.coverprofile > $(TARGET_DIR)/cover.out
	@go tool cover -html=$(TARGET_DIR)/cover.out -o=$(TARGET_DIR)/cover.html

test-xunit-report: $(GO_GO2XUNIT) $(TARGET_DIR)
	@for TST in `find . -name "tests.log"`; do \
		if [ -s $$TST ]; then \
			FAILED=`grep -E '(FAIL)+\s([a-z\.\/]*)\s\[build failed\]' $$TST | wc -l`; \
			if [ $$FAILED -gt 0 ]; then \
				echo "Build Failed \t\t\t($$TST)"; \
				echo "Build Failed \t\t\t($$TST)" >>  $(TARGET_DIR)/fail; \
			else \
				NO_TESTS=`grep -E '\?+\s+([a-z\.\/]*)\s\[no test files\]' $$TST | wc -l`; \
				if [ $$NO_TESTS -gt 0 ]; then \
					echo "No tests found \t\t\t($$TST)"; \
				else \
					echo "Generating xUnit report \t$$TST.xml $(GO_GO2XUNIT) -input $$TST -output $$TST.xml"; \
					$(GO_GO2XUNIT) -input $$TST -output $$TST.xml; \
				fi; \
			fi; \
		else \
			echo "Ignoring empty file \t\t$$TST"; \
		fi; \
	done; \
	for XML in `find . -name "tests.log.xml"`; do \
		if [ "$$XML" =  "./tests.log.xml" ]; then \
      PWD=`pwd`; \
		 	mv $$XML $(TARGET_DIR)/`basename $(PWD)`.xml; \
		else \
			mv $$XML $(TARGET_DIR)/`echo $$XML | sed 's|./||' | sed 's|/|_|g' | sed 's|_tests.log||'`; \
		fi; \
	done; \
	if [ -e $(TARGET_DIR)/fail ]; then \
		echo "ERROR: Test compilation failure"; \
		cat $(TARGET_DIR)/fail; \
		exit 1; \
	fi;

lint: $(GO_LINT)
	$(GO_LINT) --install --force
	$(GO_LINT) --fast --tests --vendor --disable=gas --disable=gotype ./...
