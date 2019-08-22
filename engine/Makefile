.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)
GITHASH := $(if ${GIT_HASH},${GIT_HASH},`git log -1 --format="%H"`)
BUILDTIME := `date "+%m/%d/%y-%H:%M:%S"`

TARGET_DIR = ./dist
TARGET_ENGINE = cds-engine
DBMIGRATE = $(words $(wildcard sql/*.sql))
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION) -X github.com/ovh/cds/sdk.GOOS=$$GOOS -X github.com/ovh/cds/sdk.GOARCH=$$GOARCH -X github.com/ovh/cds/sdk.GITHASH=$(GITHASH) -X github.com/ovh/cds/sdk.BUILDTIME=$(BUILDTIME) -X github.com/ovh/cds/sdk.BINARY=$(TARGET_ENGINE) -X github.com/ovh/cds/sdk.DBMIGRATE=$(DBMIGRATE)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386 arm64)

GO_BUILD = go build
GO_LINT = ${GOPATH}/bin/gometalinter
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GOJUNIT = ${GOPATH}/bin/go-junit-report
GO_COBERTURA = ${GOPATH}/bin/gocover-cobertura

PKGS_COMMA_SEP = go list -f '{{ join .Deps "\n" }}' . | grep github.com/ovh/cds | grep -v vendor | tr '\n' ',' | sed 's/,$$//'

TEST_PKGS = $(shell go list ../... | grep -v vendor | grep -v github.com/ovh/cds/cli/cds | grep -v github.com/ovh/cds/contrib)
TEST_DB_HOST = $(if ${CDS_API_DATABASE_HOST},${CDS_API_DATABASE_HOST},localhost)
TEST_DB_PORT = $(if ${CDS_API_DATABASE_PORT},${CDS_API_DATABASE_PORT},5432)
TEST_DB_USER = $(if ${CDS_API_DATABASE_USER},${CDS_API_DATABASE_USER},cds)
TEST_DB_PASSWORD = $(if ${CDS_API_DATABASE_PASSWORD},${CDS_API_DATABASE_PASSWORD},cds)
TEST_DB_NAME = $(if ${CDS_API_DATABASE_NAME},${CDS_API_DATABASE_NAME},cds)
TEST_REDIS_HOST = $(if ${CDS_API_CACHE_REDIS_HOST},${CDS_API_CACHE_REDIS_HOST},localhost:6379)
TEST_REDIS_PASSWORD = $(if ${CDS_API_CACHE_REDIS_PASSWORD},${CDS_API_CACHE_REDIS_PASSWORD},)
TEST_DB_START = docker run -d -p $(TEST_DB_PORT):5432 -e POSTGRES_PASSWORD=$(TEST_DB_PASSWORD) -e POSTGRES_USER=$(TEST_DB_USER) -e POSTGRES_DB=$(TEST_DB_NAME) --name postgres-cds postgres
TEST_DB_INIT = $(GO_BUILD) && ./engine database upgrade --db-host $(TEST_DB_HOST) --db-port $(TEST_DB_PORT) --db-user $(TEST_DB_USER) --db-password $(TEST_DB_PASSWORD) --db-name $(TEST_DB_NAME) --db-sslmode disable --migrate-dir ./sql && rm -rf ./engine
TEST_CMD = go test -v -timeout 600s -coverprofile=profile.coverprofile

$(TARGET_DIR):
	$(info create $(TARGET_DIR) directory)
	@mkdir -p $(TARGET_DIR)

$(GO_LINT):
	go get -u github.com/alecthomas/gometalinter

$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

$(GO_GOJUNIT):
	go get -u github.com/jstemmer/go-junit-report

$(GO_COBERTURA):
	go get -u github.com/t-yuki/gocover-cobertura

default: build

clean:
	@rm -rf $(TARGET_DIR)
	@for TST in `find ../ -name "tests.log"`; do \
		rm $$TST; \
	done;
	@for profile in `find ../ -name "*.coverprofile"`; do \
		rm $$profile; \
	done;

build: $(TARGET_DIR)
	@for GOOS in $(TARGET_OS); do \
		for GOARCH in $(TARGET_ARCH); do \
			EXTENSION=""; \
			if test "$$GOOS" = "windows" ; then EXTENSION=".exe"; fi; \
			echo Compiling $(TARGET_DIR)/$(TARGET_ENGINE)-$$GOOS-$$GOARCH$$EXTENSION $(VERSION); \
			GOOS=$$GOOS GOARCH=$$GOARCH $(GO_BUILD) $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$(TARGET_ENGINE)-$$GOOS-$$GOARCH$$EXTENSION; \
		done; \
	done
	@tar czf dist/sql.tar.gz sql/*
	$(MAKE) build -C worker

test-db-start:
	$(TEST_DB_START)
	@sleep 5
	$(TEST_DB_INIT)

test-db-init:
	$(TEST_DB_INIT)

test-config: ${HOME}/.cds/tests.cfg.json

${HOME}/.cds/tests.cfg.json:
		@mkdir -p ${HOME}/.cds
		@echo "{\"dbDriver\": \"postgres\", \"dbUser\" :\"$(TEST_DB_USER)\",  \"dbPassword\" :\"$(TEST_DB_PASSWORD)\"  ,\"dbName\": \"$(TEST_DB_NAME)\", \"dbHost\": \"$(TEST_DB_HOST)\", \"dbPort\" : \"$(TEST_DB_PORT)\", \"sslMode\": \"disable\", \"redisHost\" : \"$(TEST_REDIS_HOST)\", \"redisPassword\" : \"$(TEST_REDIS_PASSWORD)\"}" > ${HOME}/.cds/tests.cfg.json

test: test-config clean
	@for PKG in $(TEST_PKGS); do \
		echo "Running tests in package $$PKG"; \
		cd ${GOPATH}/src/$$PKG; \
		TEMP=`$(PKGS_COMMA_SEP)`; \
    if [ ! -z "$$TEMP" ]; then \
		  $(TEST_CMD) -coverpkg $$TEMP > ${GOPATH}/src/$$PKG/tests.log; \
    else \
		  $(TEST_CMD) > ${GOPATH}/src/$$PKG/tests.log; \
    fi; \
	done;
	$(MAKE) test-coverage
	$(MAKE) test-xunit-report

test-coverage: $(GO_COV_MERGE) $(TARGET_DIR) $(GO_COBERTURA)
	@echo "Generating unit tests coverage..."
	@$(GO_COV_MERGE) `find ../ -name "*.coverprofile"` > $(TARGET_DIR)/cover.out
	@$(GO_COBERTURA) < $(TARGET_DIR)/cover.out > $(TARGET_DIR)/coverage.xml
	@go tool cover -html=$(TARGET_DIR)/cover.out -o=$(TARGET_DIR)/cover.html
	@if [ ! -z "${CDS_VERSION}" ]; then \
		echo "Sending coverage report to CDS"; \
		worker upload --tag "${CDS_VERSION}" $(abspath $(TARGET_DIR)/cover.html); \
	fi;

#  <testsuite name="github.com/ovh/cds/engine/api" tests="133" errors="0" failures="2" skip="0">
test-xunit-report: $(GO_GOJUNIT) $(TARGET_DIR)
	@echo "Generating xUnit Report..."
	@for TST in `find .. -name "tests.log"`; do \
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
					if [ ! -z "${CDS_VERSION}" ]; then \
						echo "Sending $$TST to CDS"; \
						worker upload --tag `echo $$TST | sed 's|../||' | sed 's|./||' | sed 's|/|_|g') | sed 's|_tests.log||'` $(abspath $$TST); \
					fi; \
					echo "Generating xUnit report \t$$TST.xml"; \
					cat $$TST | $(GO_GOJUNIT) > $$TST.xml; \
				fi; \
			fi; \
		else \
			echo "Ignoring empty file \t\t$$TST"; \
		fi; \
	done; \
	for XML in `find .. -name "tests.log.xml"`; do \
		if [ "$$XML" =  "./tests.log.xml" ]; then \
      		PWD=`pwd`; \
		 	mv $$XML $(TARGET_DIR)/`basename $(PWD)`.xml; \
		else \
			mv $$XML $(TARGET_DIR)/`echo $$XML | sed 's|../||' | sed 's|/|_|g' | sed 's|_tests.log||'`; \
		fi; \
	done; \
	for XML in `find .. -name "*.xml" -not -path "../ui/*"`; do \
		if [ -s $$XML ]; then \
			if grep -q 'name' $$XML; then \
				echo "Generating report: " $$XML; \
				echo "`xmllint --xpath "//testsuite/@name" $$XML | sed 's/name=//' | sed 's/"//g'`" \
				"`xmllint --xpath "//testsuite/@tests" $$XML | sed 's/tests=//' | sed 's/"//g'` Tests :" \
				"`xmllint --xpath "//testsuite/@errors" $$XML | sed 's/errors=//' | sed 's/"//g'` Errors ;"\
				"`xmllint --xpath "//testsuite/@failures" $$XML | sed 's/failures=//' | sed 's/"//g'` Failures;" \
				"`xmllint --xpath "//testsuite/@skip" $$XML | sed 's/skip=//' | sed 's/"//g'` Skipped;" \
				>> $(TARGET_DIR)/report; \
			fi; \
		fi; \
	done; \
	if [ -e $(TARGET_DIR)/report ]; then \
		cat $(TARGET_DIR)/report; \
	fi; \
	echo "#########################"; \
	for XML in `find .. -name "*.xml" -not -path "../ui/*"`; do \
		if [ -s $$XML ]; then \
			if grep -q 'errors' $$XML; then \
				if [ "`xmllint --xpath "//testsuite/@errors" $$XML | sed 's/errors=//' | sed 's/"//g'`" -gt "0" ]; then  \
					echo "	$$XML : Tests failed";  \
				fi; \
			fi; \
			if grep -q 'failures' $$XML; then \
				if [ "`xmllint --xpath "//testsuite/@failures" $$XML | sed 's/failures=//' | sed 's/"//g'`" -gt "0" ]; then  \
					echo "	$$XML : Tests failed";  \
				fi; \
			fi; \
		fi; \
	done; \
	if [ -e $(TARGET_DIR)/fail ]; then \
		echo "#########################"; \
		echo "ERROR: Test compilation failure"; \
		cat $(TARGET_DIR)/fail; \
		exit 1; \
	fi;

lint: $(GO_LINT)
	$(GO_LINT) --install --force
	$(GO_LINT) --fast --tests --vendor --disable=gas --disable=gotype ./...