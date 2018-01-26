.PHONY: clean

VERSION := $(if ${CDS_SEMVER},${CDS_SEMVER},snapshot)

TARGET_DIR = ./dist
TARGET_ENGINE = cds-engine
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm 386)

GO_BUILD = go build -v
GO_LINT = ${GOPATH}/bin/gometalinter
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GOVERALLS = ${GOPATH}/bin/goveralls
GO_GO2XUNIT = ${GOPATH}/bin/go2xunit

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
		$(TEST_CMD) > ${GOPATH}/src/$$PKG/tests.log; \
	done;
	$(MAKE) test-coverage
	$(MAKE) test-xunit-report

test-coverage: $(GO_GOVERALLS) $(GO_COV_MERGE) $(TARGET_DIR)
	@$(GO_COV_MERGE) ./**/*.coverprofile > $(TARGET_DIR)/cover.out
	@go tool cover -html=$(TARGET_DIR)/cover.out -o=$(TARGET_DIR)/cover.html
ifneq ($(VERSION), snapshot)
	echo "Pushing results to coveralls"
	@$(GO_GOVERALLS) -coverprofile=$(TARGET_DIR)/cover.out -service=cds -repotoken ${CDS_PROJ_COVERALLS_TOKEN}
endif

#  <testsuite name="github.com/ovh/cds/engine/api" tests="133" errors="0" failures="2" skip="0">
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
					echo "Generating xUnit report \t$$TST.xml"; \
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
	for XML in `find . -name "*.xml"`; do \
		if [ -s $$XML ]; then \
			echo "`xmllint --xpath "/testsuite/@name" $$XML | sed 's/name=//' | sed 's/"//g'`" \
			"`xmllint --xpath "/testsuite/@tests" $$XML | sed 's/tests=//' | sed 's/"//g'` Tests :" \
			"`xmllint --xpath "/testsuite/@errors" $$XML | sed 's/errors=//' | sed 's/"//g'` Errors "\
			"`xmllint --xpath "/testsuite/@failures" $$XML | sed 's/failures=//' | sed 's/"//g'` Failures" \
			"`xmllint --xpath "/testsuite/@skip" dist/api.xml | sed 's/skip=//' | sed 's/"//g'` Skipped" \
			>> $(TARGET_DIR)/report; \
		fi; \
	done; \
	if [ -e $(TARGET_DIR)/report ]; then \
		cat $(TARGET_DIR)/report; \
	fi; \
	for XML in `find . -name "*.xml"`; do \
		if [ -s $$XML ]; then \
			if [ "`xmllint --xpath "/testsuite/@errors" $$XML | sed 's/errors=//' | sed 's/"//g'`" -gt "0" ]; then  \
				echo "Tests failed";  \
				exit 1;  \
			fi; \
			if [ "`xmllint --xpath "/testsuite/@failures" $$XML | sed 's/failures=//' | sed 's/"//g'`" -gt "0" ]; then  \
				echo "Tests failed";  \
				exit 1;  \
			fi; \
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
