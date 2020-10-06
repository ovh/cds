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
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GOJUNIT = ${GOPATH}/bin/go-junit-report
GO_COBERTURA = ${GOPATH}/bin/gocover-cobertura
GO_XUTOOLS = ${GOPATH}/bin/xutools

PKGS_COMMA_SEP = go list -f '{{ join .Deps "\n" }}{{"\n"}}{{.ImportPath}}' . | grep github.com/ovh/cds | grep -v vendor | tr '\n' ',' | sed 's/,$$//'

TEST_PKGS = $(shell go list ./...)

TEST_DB_HOST = $(if ${CDS_DATABASE_HOST},${CDS_DATABASE_HOST},localhost)
TEST_DB_PORT = $(if ${CDS_DATABASE_PORT},${CDS_DATABASE_PORT},5432)
TEST_DB_USER = $(if ${CDS_DATABASE_USER},${CDS_DATABASE_USER},cds)
TEST_DB_PASS = $(if ${CDS_DATABASE_PASS},${CDS_DATABASE_PASS},cds)
TEST_DB_NAME = $(if ${CDS_DATABASE_NAME},${CDS_DATABASE_NAME},cds)
TEST_DB_DROP_DATABASE = PGPASSWORD=${TEST_DB_PASS} psql -h ${TEST_DB_HOST} -p ${TEST_DB_PORT} -U ${TEST_DB_USER} -d postgres -c "drop database if exists ${TEST_DB_NAME}"
TEST_DB_CREATE_DATABASE = PGPASSWORD=${TEST_DB_PASS} psql -h ${TEST_DB_HOST} -p ${TEST_DB_PORT} -U ${TEST_DB_USER} -d postgres -c "create database ${TEST_DB_NAME} owner ${TEST_DB_USER}"

TEST_API_DB_SCHEMA = $(if ${CDS_API_DATABASE_SCHEMA},${CDS_API_DATABASE_SCHEMA},public)
TEST_API_DB_CREATE_SCHEMA = PGPASSWORD=${TEST_DB_PASS} psql -h ${TEST_DB_HOST} -p ${TEST_DB_PORT} -U ${TEST_DB_USER} -d ${TEST_DB_USER} -c "CREATE SCHEMA IF NOT EXISTS ${TEST_API_DB_SCHEMA} AUTHORIZATION ${TEST_DB_USER};"
TEST_API_DB_INIT = ./engine database upgrade --db-host $(TEST_DB_HOST) --db-port $(TEST_DB_PORT) --db-user $(TEST_DB_USER) --db-password $(TEST_DB_PASS) --db-name $(TEST_DB_NAME) --db-schema $(TEST_API_DB_SCHEMA) --db-sslmode disable --migrate-dir ./sql/api

TEST_CDN_DB_SCHEMA = $(if ${CDS_CDN_DATABASE_SCHEMA},${CDS_CDN_DATABASE_SCHEMA},cdn)
TEST_CDN_DB_CREATE_SCHEMA = PGPASSWORD=${TEST_DB_PASS} psql -h ${TEST_DB_HOST} -p ${TEST_DB_PORT} -U ${TEST_DB_USER} -d ${TEST_DB_USER} -c "CREATE SCHEMA IF NOT EXISTS ${TEST_CDN_DB_SCHEMA} AUTHORIZATION ${TEST_DB_USER};"
TEST_CDN_DB_INIT = ./engine database upgrade --db-host $(TEST_DB_HOST) --db-port $(TEST_DB_PORT) --db-user $(TEST_DB_USER) --db-password $(TEST_DB_PASS) --db-name $(TEST_DB_NAME) --db-schema $(TEST_CDN_DB_SCHEMA) --db-sslmode disable --migrate-dir ./sql/cdn

TEST_DB_STOP_DOCKER = docker rm -f postgres-cds
TEST_DB_START_DOCKER = docker run -d -p $(TEST_DB_PORT):5432 -e POSTGRES_PASSWORD=$(TEST_DB_PASS) -e POSTGRES_USER=$(TEST_DB_USER) -e POSTGRES_DB=$(TEST_DB_NAME) --name postgres-cds postgres

TEST_REDIS_PORT = 6379
TEST_REDIS_HOST = $(if ${CDS_CACHE_REDIS_HOST},${CDS_CACHE_REDIS_HOST},localhost:$(TEST_REDIS_PORT))
TEST_REDIS_PASS = $(if ${CDS_CACHE_REDIS_PASS},${CDS_CACHE_REDIS_PASS},)
TEST_REDIS_START_DOCKER = docker run -d -p $(TEST_REDIS_PORT):6379 --name redis-cds redis

TEST_KAFKA_START_DOCKER = docker run -d -p 2181:2181 -p 9092:9092 --env ADVERTISED_HOST=localhost --env ADVERTISED_PORT=9092 --name kafka-cds spotify/kafka

TEST_CMD = go test -tags skipkeychaintests -v -timeout 600s -coverprofile=profile.coverprofile

$(TARGET_DIR):
	@mkdir -p $(TARGET_DIR)

$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

$(GO_GOJUNIT):
	go get -u github.com/jstemmer/go-junit-report

$(GO_COBERTURA):
	go get -u github.com/t-yuki/gocover-cobertura

$(GO_XUTOOLS):
	go get -u github.com/richardlt/xutools

default: build

clean:
	@rm -rf $(TARGET_DIR)
	@for TST in `find ./ -name "tests.log"`; do \
		rm $$TST; \
	done;
	@for profile in `find ./ -name "*.coverprofile"`; do \
		rm $$profile; \
	done;

build: $(TARGET_DIR)
	$(MAKE) lint -C sql
	$(MAKE) build -C sql
	@cp sql/sql.tar.gz $(TARGET_DIR)/
	$(MAKE) build_all_engine
	$(MAKE) build -C worker
define get_os_from_binary_file
$(strip $(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n-1]}'))
endef

define get_arch_from_binary_file
$(strip $(patsubst %.exe, %,$(shell echo $(1) | awk '{n=split($$1,a,"-");print a[n]}')))
endef

gobuild:
	$(info Checking support for $(GOOS)/$(GOARCH))
	@if go tool dist list | grep "$$GOOS" | grep "$$GOARCH" > /dev/null 2>&1; then \
		GOOS=$$GOOS GOARCH=$$GOARCH CGO_ENABLED=0 $(GO_BUILD) $(TARGET_LDFLAGS) -o $(abspath $(OUTPUT)); \
	fi

isWindows = $(filter $1,windows)
TARGET_ENGINE_BINARIES := $(foreach OS, $(TARGET_OS), $(foreach ARCH, $(TARGET_ARCH), $(TARGET_DIR)/$(TARGET_ENGINE)-$(OS)-$(ARCH)$(if $(call isWindows, $(OS)),.exe)))
$(TARGET_ENGINE_BINARIES): $(TARGET_DIR)
	$(info Compiling $@: $(abspath $@))
	$(MAKE) --no-print-directory gobuild GOOS=$(call get_os_from_binary_file,$@) GOARCH=$(call get_arch_from_binary_file,$@) OUTPUT=$@

build_all_engine: $(TARGET_ENGINE_BINARIES)

test-redis-start-docker:
	$(TEST_REDIS_START_DOCKER)

test-kafka-start-docker:
	$(TEST_KAFKA_START_DOCKER)

test-db-stop-docker:
	$(TEST_DB_STOP_DOCKER)

test-db-start-docker:
	$(TEST_DB_START_DOCKER)
	@sleep 5
	$(MAKE) test-api-db-create
	$(MAKE) test-cdn-db-create

test-api-db-create:
	$(TEST_API_DB_CREATE_SCHEMA)
	$(TEST_API_DB_INIT)

test-cdn-db-create:
	$(TEST_CDN_DB_CREATE_SCHEMA)
	$(TEST_CDN_DB_INIT)

test-db-recreate:
	$(TEST_DB_DROP_DATABASE)
	$(TEST_DB_CREATE_DATABASE)
	$(TEST_API_DB_CREATE_SCHEMA)
	$(TEST_API_DB_INIT)
	$(TEST_CDN_DB_CREATE_SCHEMA)
	$(TEST_CDN_DB_INIT)

test-db-init: test-config
	$(MAKE) test-api-db-create
	$(MAKE) test-cdn-db-create

test-config: ${HOME}/.cds/api.tests.cfg.json ${HOME}/.cds/cdn.tests.cfg.json

${HOME}/.cds/api.tests.cfg.json:
		@mkdir -p ${HOME}/.cds
		@echo "{\"dbDriver\": \"postgres\", \"dbUser\" :\"$(TEST_DB_USER)\", \"dbPassword\" :\"$(TEST_DB_PASS)\" ,\"dbName\": \"$(TEST_DB_NAME)\", \"dbSchema\": \"$(TEST_API_DB_SCHEMA)\", \"dbHost\": \"$(TEST_DB_HOST)\", \"dbPort\" : \"$(TEST_DB_PORT)\", \"sslMode\": \"disable\", \"redisHost\" : \"$(TEST_REDIS_HOST)\", \"redisPassword\" : \"$(TEST_REDIS_PASSWORD)\"}" > ${HOME}/.cds/api.tests.cfg.json

${HOME}/.cds/cdn.tests.cfg.json:
		@mkdir -p ${HOME}/.cds
		@echo "{\"dbDriver\": \"postgres\", \"dbUser\" :\"$(TEST_DB_USER)\", \"dbPassword\" :\"$(TEST_DB_PASS)\" ,\"dbName\": \"$(TEST_DB_NAME)\", \"dbSchema\": \"$(TEST_CDN_DB_SCHEMA)\", \"dbHost\": \"$(TEST_DB_HOST)\", \"dbPort\" : \"$(TEST_DB_PORT)\", \"sslMode\": \"disable\", \"redisHost\" : \"$(TEST_REDIS_HOST)\", \"redisPassword\" : \"$(TEST_REDIS_PASSWORD)\"}" > ${HOME}/.cds/cdn.tests.cfg.json

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
	@$(GO_COV_MERGE) `find ./ -name "*.coverprofile"` > $(TARGET_DIR)/cover.out
	@$(GO_COBERTURA) < $(TARGET_DIR)/cover.out > $(TARGET_DIR)/coverage.xml
	@go tool cover -html=$(TARGET_DIR)/cover.out -o=$(TARGET_DIR)/cover.html

#  <testsuite name="github.com/ovh/cds/engine/api" tests="133" errors="0" failures="2" skip="0">
test-xunit-report: $(GO_GOJUNIT) $(GO_XUTOOLS) $(TARGET_DIR)
	@echo "Generating xUnit Report..."
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
					if [ "$$TST" = "./tests.log" ]; then \
						PACKAGE=engine; \
					else \
						PACKAGE=engine_`echo $$TST | sed 's|./||' | sed 's|/|_|g' | sed 's|_tests.log||'`; \
					fi; \
					XUNIT_OUT=$(TARGET_DIR)/$$PACKAGE.xml; \
					echo "Generating xUnit report from \t$$TST.xml to $$XUNIT_OUT"; \
					cat $$TST | $(GO_GOJUNIT) > $$XUNIT_OUT; \
					TESTS_LOG_OUT=$(TARGET_DIR)/$$PACKAGE.log; \
					echo "Copy tests log from \t$$TST.xml to $$TESTS_LOG_OUT"; \
					cp $$TST $$TESTS_LOG_OUT; \
				fi; \
			fi; \
		else \
			echo "Ignoring empty file \t\t$$TST"; \
		fi; \
	done; \
	xutools pretty --show-failures ./dist/*.xml > $(TARGET_DIR)/report; \
	xutools sort-duration ./dist/*.xml > $(TARGET_DIR)/duration; \
	if [ -e $(TARGET_DIR)/report ]; then \
		echo "Report:"; \
		cat $(TARGET_DIR)/report; \
	fi; \
	if [ -e $(TARGET_DIR)/duration ]; then \
		echo "Max duration:"; \
		cat $(TARGET_DIR)/duration; \
	fi; \
	if [ -e $(TARGET_DIR)/fail ]; then \
		echo "#########################"; \
		echo "ERROR: Test compilation failure"; \
		cat $(TARGET_DIR)/fail; \
		exit 1; \
	fi;
