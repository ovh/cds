VERSION = snapshot

TARGET_DIR = dist
TARGET_ENGINE = cds-engine
TARGET_LDFLAGS = -ldflags "-X github.com/ovh/cds/sdk.VERSION=$(VERSION)"
TARGET_OS = windows darwin linux freebsd
TARGET_ARCH = amd64 arm

GO_LINT = ${GOPATH}/bin/gometalinter
GO_COV_MERGE = ${GOPATH}/bin/gocovmerge
GO_GOVERALLS = ${GOPATH}/bin/goveralls

TEST_PKGS = $(shell go list ./... | grep -v vendor)
TEST_DB_HOST = localhost
TEST_DB_PORT = 5432
TEST_DB_USER = cds
TEST_DB_PASSWORD = cds
TEST_DB_NAME = cds
TEST_DB_START = docker run -d -p $(TEST_DB_PORT):5432 -e POSTGRES_PASSWORD=$(TEST_DB_PASSWORD) -e POSTGRES_USER=$(TEST_DB_USER) -e POSTGRES_DB=$(TEST_DB_NAME) --name postgres-cds postgres
TEST_DB_INIT = go build && ./engine database upgrade --db-host $(TEST_DB_HOST) --db-port $(TEST_DB_PORT) --db-user $(TEST_DB_USER) --db-password $(TEST_DB_PASSWORD) --db-name $(TEST_DB_NAME) --db-sslmode disable --migrate-dir ./sql && rm -rf ./engine

.PHONY: clean

default: build

$(GO_LINT):
	go get -u github.com/alecthomas/gometalinter

$(GO_COV_MERGE):
	go get -u github.com/wadey/gocovmerge

$(GO_GOVERALLS):
	go get -u github.com/mattn/goveralls


build:
	for GOOS in $(TARGET_OS); do for GOARCH in $(TARGET_ARCH); $(info Compiling $(TARGET_DIR)/$(TARGET_ENGINE)-$$GOOS-$$GOARCH); go build $(TARGET_LDFLAGS) -o $(TARGET_DIR)/$(TARGET_ENGINE)-$$GOOS-$$GOARCH; done; done

test-db-start:
	$(TEST_DB_START)
	sleep 5
	$(TEST_DB_INIT)

test-config:
	echo "{\"dbDriver\": \"postgres\", \"dbUser\" :\"$(TEST_DB_USER)\",  \"dbPassword\" :\"$(TEST_DB_PASSWORD)\"  ,\"dbName\": \"$(TEST_DB_NAME)\", \"dbHost\", \"$(TEST_DB_HOST)\", \"dbPort\" : \"$(TEST_DB_PORT)\", \"sslMode\": \"disable\"}" > ${HOME}/.cds/tests.cfg.json

test:
	$(foreach PKG,$(TEST_PKGS), $(shell cd ${GOPATH}/src/$(PKG) && go test -v -timeout 180s -coverprofile=profile.coverprofile > ${GOPATH}/src/$(PKG)/tests.log))
	$(GO_COV_MERGE) ./**/*.coverprofile > dist/cover.out
	go tool cover -html=dist/cover.out -o=dist/cover.html

clean:
	rm -rf dist/
	rm -f ./tests.log **/tests.log
	rm -f ./*.coverprofile **/*.coverprofile


lint:
	$(GO_LINT) --install --force
	$(GO_LINT) --fast --tests --vendor --disable=gas --disable=gotype ./...