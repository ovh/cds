.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)

TARGET_DIR = ./dist
TARGET_BINARY = cds-worker
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

GO_BUILD = go build
GO_LINT = ${GOPATH}/bin/gometalinter
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GOVERALLS = ${GOPATH}/bin/goveralls
GO_GO2XUNIT = ${GOPATH}/bin/go2xunit

TEST_PKGS = $(shell go list ./... | grep -v vendor)
TEST_CMD = go test -v -timeout 180s -coverprofile=profile.coverprofile

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

$(GO_LINT):
	go get -u github.com/alecthomas/gometalinter

$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

$(GO_GOVERALLS):
	go get -u github.com/mattn/goveralls

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

build: clean $(TARGET_DIR)
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			echo Compiling $(TARGET_DIR)/$(TARGET_BINARY)-$$GOOS-$$GOARCH $(VERSION); \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$(TARGET_BINARY)-$$GOOS-$$GOARCH; \
		done; \
	done

test: clean
	@for PKG in $(TEST_PKGS); do \
		echo "Running tests in package $$PKG"; \
		cd ${GOPATH}/src/$$PKG; \
		$(TEST_CMD) > ${GOPATH}/src/$$PKG/tests.log; \
	done;
	$(MAKE) test-coverage
	$(MAKE) test-xunit-report

test-coverage: $(GO_GOVERALLS) $(GO_COV_MERGE) $(TARGET_DIR)
	@$(GO_COV_MERGE) ./*.coverprofile > $(TARGET_DIR)/cover.out
	@go tool cover -html=$(TARGET_DIR)/cover.out -o=$(TARGET_DIR)/cover.html
ifneq ($(VERSION), snapshot)
	echo "Pushing results to coveralls"
	@$(GO_GOVERALLS) -coverprofile=$(TARGET_DIR)/cover.out -service=cds -repotoken ${CDS_PROJ_COVERALLS_TOKEN}
endif

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
		 	mv $$XML $(TARGET_DIR)/`basename `pwd``.xml; \
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
