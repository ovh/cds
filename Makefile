.PHONY: clean doc modclean mod goinstall build dist deb watch watch-deps watch-init watch-conf watch-db watch-ui watch-ui-rebuild watch-plugins watch-setup-forgejo

TARGET_OS = $(if ${OS},${OS},windows darwin linux freebsd)
TARGET_ARCH = $(if ${ARCH},${ARCH},amd64 arm64)
VERSION := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
GIT_DESCRIBE := $(shell git describe --tags)
GIT_VERSION := $(if ${GIT_DESCRIBE},${GIT_DESCRIBE:v%=%},0.0.0-0-snapshot)
SHA512 := $(if ifeq ${UNAME} "Darwin",shasum -a 512,sha512sum)

TARGET_ENGINE := $(if ${TARGET_ENGINE},${TARGET_ENGINE},engine)
TARGET_WORKER := $(if ${TARGET_WORKER},${TARGET_WORKER},worker)
TARGET_CDSCTL := $(if ${TARGET_CDSCTL},${TARGET_CDSCTL},cdsctl)

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

build:
	$(info Building CDS Components for $(TARGET_OS) - $(TARGET_ARCH))
	$(MAKE) build_ui -j1
	$(MAKE) embed_ui
	$(MAKE) build_engine -j4
	$(MAKE) build_worker -j4
	$(MAKE) build_cli -j4
	$(MAKE) build_contrib -j4
	$(MAKE) package -C contrib TARGET_DIST="$(abspath $(TARGET_DIR))"
	$(MAKE) archive -C contrib TARGET_DIST="$(abspath $(TARGET_DIR))"

build_ui:
	$(MAKE) build -C ui

embed_ui:
	@echo "Embedding UI static files into engine/ui/dist/"
	@if [ -d ui/dist/browser ]; then \
		rm -rf engine/ui/dist; \
		mkdir -p engine/ui/dist; \
		cp -r ui/dist/browser/* engine/ui/dist/; \
		echo "UI files embedded successfully"; \
	else \
		echo "No UI build found at ui/dist/browser/, using .gitkeep placeholder"; \
	fi

build_engine:
	$(MAKE) build -C engine OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

build_worker:
	$(MAKE) build -C engine/worker OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

build_cli:
	$(MAKE) build -C cli/cdsctl OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

build_contrib:
	$(MAKE) build -C contrib OS="${TARGET_OS}" ARCH="${TARGET_ARCH}"

define get_dist_from_target
$(filter %/$(notdir $(1)), $(ALL_DIST))
endef

$(ALL_TARGETS):
	@mkdir -p $(TARGET_DIR)
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

tidy:
	@echo "Running tidy on cds main project"
	@go mod tidy
	@echo "Running tidy on contrib/grpcplugins/action/"
	$(MAKE) tidy -C contrib/grpcplugins/action

	@echo "Running tidy on contrib/integrations/"
	$(MAKE) tidy -C contrib/integrations

	@echo "Running tidy on tests/fixtures/04SCWorkflowRunSimplePlugin"
	@(cd tests/fixtures/04SCWorkflowRunSimplePlugin && go mod tidy)
	@(cd sdk/interpolate && go mod tidy)
	@(cd tools/smtpmock && go mod tidy)

# ==============================================================================
# Local development — `make watch`
#
# Starts external dependencies in docker, builds the engine, and runs all
# CDS services in a single process with in-process communication.
#
# First run:
#   make watch-deps   (start postgres, redis, opensearch)
#   make watch-init   (create schemas, run migrations, generate config)
#   make watch        (build + run)
#
# After that, just:
#   make watch        (rebuild + restart)
#
# ==============================================================================

DEV_PG_CONTAINER    ?= cds-dev-postgres
DEV_REDIS_CONTAINER ?= cds-dev-redis
DEV_OS_CONTAINER    ?= cds-dev-opensearch
DEV_FORGEJO_CONTAINER ?= cds-dev-forgejo
DEV_CONF            ?= $(HOME)/.cds/dev.conf.toml
DEV_DIR             ?= $(HOME)/.cds/dev

DEV_DB_HOST     ?= localhost
DEV_DB_PORT     ?= 5432
DEV_DB_USER     ?= cds
DEV_DB_PASS     ?= cds
DEV_DB_NAME     ?= cds
DEV_REDIS_PORT  ?= 6379
DEV_REDIS_PASS  ?= cds
DEV_OS_PORT     ?= 9200
DEV_FORGEJO_PORT ?= 3000
DEV_FORGEJO_SSH_PORT ?= 2222

DEV_GOOS    := $(shell go env GOOS)
DEV_GOARCH  := $(shell go env GOARCH)
ENGINE_BIN   = engine/dist/cds-engine-$(DEV_GOOS)-$(DEV_GOARCH)
WORKER_BIN   = engine/worker/dist/cds-worker-$(DEV_GOOS)-$(DEV_GOARCH)
CDSCTL_BIN   = cli/cdsctl/dist/cdsctl-$(DEV_GOOS)-$(DEV_GOARCH)

DEV_VERSION  := $(if ${CDS_VERSION},${CDS_VERSION},snapshot)
DEV_GITHASH  := $(shell git log -1 --format="%H" 2>/dev/null || echo "unknown")
DEV_DBMIGRATE:= $(words $(wildcard engine/sql/*.sql))
DEV_LDFLAGS   = -ldflags "\
	-X github.com/ovh/cds/sdk.VERSION=$(DEV_VERSION) \
	-X github.com/ovh/cds/sdk.GOOS=$(DEV_GOOS) \
	-X github.com/ovh/cds/sdk.GOARCH=$(DEV_GOARCH) \
	-X github.com/ovh/cds/sdk.GITHASH=$(DEV_GITHASH) \
	-X github.com/ovh/cds/sdk.BUILDTIME=$$(date '+%m/%d/%y-%H:%M:%S') \
	-X github.com/ovh/cds/sdk.BINARY=cds-engine \
	-X github.com/ovh/cds/sdk.DBMIGRATE=$(DEV_DBMIGRATE)"

# Start docker containers for postgres, redis, opensearch
watch-deps:
	@echo "▶ Starting dev dependencies..."
	@docker start $(DEV_PG_CONTAINER) 2>/dev/null || \
		docker run -d --name $(DEV_PG_CONTAINER) \
			-p $(DEV_DB_PORT):5432 \
			-e POSTGRES_PASSWORD=$(DEV_DB_PASS) \
			-e POSTGRES_USER=$(DEV_DB_USER) \
			-e POSTGRES_DB=$(DEV_DB_NAME) \
			postgres:14.0
	@docker start $(DEV_REDIS_CONTAINER) 2>/dev/null || \
		docker run -d --name $(DEV_REDIS_CONTAINER) \
			-p $(DEV_REDIS_PORT):6379 \
			redis:alpine redis-server --requirepass $(DEV_REDIS_PASS)
	@docker start $(DEV_OS_CONTAINER) 2>/dev/null || \
		docker run -d --name $(DEV_OS_CONTAINER) \
			-p $(DEV_OS_PORT):9200 \
			-e discovery.type=single-node \
			-e "OPENSEARCH_JAVA_OPTS=-Xms512m -Xmx512m" \
			-e DISABLE_SECURITY_PLUGIN=true \
			-e "OPENSEARCH_INITIAL_ADMIN_PASSWORD=C0mpl3xP@ssw0rd!" \
			opensearchproject/opensearch:latest
	@docker start $(DEV_FORGEJO_CONTAINER) 2>/dev/null || \
		docker run -d --name $(DEV_FORGEJO_CONTAINER) \
			-p $(DEV_FORGEJO_PORT):3000 \
			-p $(DEV_FORGEJO_SSH_PORT):2222 \
			-e FORGEJO__security__INSTALL_LOCK=true \
			-e FORGEJO__server__ROOT_URL=http://localhost:$(DEV_FORGEJO_PORT) \
			-e FORGEJO__server__SSH_PORT=$(DEV_FORGEJO_SSH_PORT) \
			codeberg.org/forgejo/forgejo:14.0-rootless
	@echo "  waiting for postgres..."
	@for i in $$(seq 1 30); do \
		docker exec $(DEV_PG_CONTAINER) pg_isready -U $(DEV_DB_USER) >/dev/null 2>&1 && break; \
		sleep 1; \
	done
	@echo "  waiting for forgejo..."
	@for i in $$(seq 1 30); do \
		curl -sf http://localhost:$(DEV_FORGEJO_PORT)/api/v1/version >/dev/null 2>&1 && break; \
		sleep 2; \
	done
	@echo "  creating forgejo admin user (idempotent)..."
	@docker exec $(DEV_FORGEJO_CONTAINER) forgejo admin user create \
		--admin --username $(DEV_ADMIN_USER) --password $(DEV_ADMIN_PASS) --email $(DEV_ADMIN_USER)@localhost.local 2>/dev/null || true
	@echo "✓ Dependencies ready (postgres, redis, opensearch, forgejo)"

# Run database migrations and create CDN schema
watch-db:
	@echo "▶ Running database migrations..."
	@docker exec $(DEV_PG_CONTAINER) psql -U $(DEV_DB_USER) -d $(DEV_DB_NAME) \
		-c "CREATE SCHEMA IF NOT EXISTS cdn AUTHORIZATION $(DEV_DB_USER);" 2>/dev/null || true
	@$(ENGINE_BIN) database upgrade \
		--db-host $(DEV_DB_HOST) --db-port $(DEV_DB_PORT) \
		--db-user $(DEV_DB_USER) --db-password $(DEV_DB_PASS) \
		--db-name $(DEV_DB_NAME) --db-schema public \
		--db-sslmode disable --migrate-dir engine/sql/api
	@$(ENGINE_BIN) database upgrade \
		--db-host $(DEV_DB_HOST) --db-port $(DEV_DB_PORT) \
		--db-user $(DEV_DB_USER) --db-password $(DEV_DB_PASS) \
		--db-name $(DEV_DB_NAME) --db-schema cdn \
		--db-sslmode disable --migrate-dir engine/sql/cdn
	@echo "✓ Database ready"

# Generate dev config if it doesn't exist
watch-conf:
	@mkdir -p $(DEV_DIR)/artifacts $(DEV_DIR)/repositories $(DEV_DIR)/hatchery-local \
		$(DEV_DIR)/cdn-buffer $(DEV_DIR)/cdn-storage $(dir $(DEV_CONF))
	@if [ ! -f $(DEV_CONF) ]; then \
		echo "▶ Generating config at $(DEV_CONF)..."; \
		$(ENGINE_BIN) config new api cdn ui hooks repositories vcs elasticsearch hatchery:local > $(DEV_CONF); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.cache.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.cache.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.database.host=$(DEV_DB_HOST); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.database.port=$(DEV_DB_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.download.directory=$(DEV_DIR); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) api.artifact.local.baseDirectory=$(DEV_DIR)/artifacts; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.cache.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.cache.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.database.host=$(DEV_DB_HOST); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.database.port=$(DEV_DB_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.storageUnits.buffers.local-buffer.local.path=$(DEV_DIR)/cdn-buffer; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.storageUnits.buffers.redis.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.storageUnits.buffers.redis.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) cdn.storageUnits.storages.local.local.path=$(DEV_DIR)/cdn-storage; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) elasticsearch.elasticsearch.url=http://$(DEV_DB_HOST):$(DEV_OS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) elasticsearch.elasticsearch.indexEvents=cds-index-events; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) elasticsearch.elasticsearch.indexEventsV2=cds-index-events-v2; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) elasticsearch.elasticsearch.indexMetrics=cds-index-metrics; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) hooks.cache.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) hooks.cache.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) repositories.basedir=$(DEV_DIR)/repositories; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) repositories.cache.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) repositories.cache.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) vcs.cache.redis.host=$(DEV_DB_HOST):$(DEV_REDIS_PORT); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) vcs.cache.redis.password=$(DEV_REDIS_PASS); \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) hatchery.local.basedir=$(DEV_DIR)/hatchery-local; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) ui.enableServiceProxy=true; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) ui.url=http://localhost:8081; \
		$(ENGINE_BIN) config edit $(DEV_CONF) --output $(DEV_CONF) ui.http.port=8081; \
		sed -i '' 's|http://localhost:8080|http://localhost:8081|g' $(DEV_CONF); \
		echo "✓ Config generated"; \
		echo ""; \
		INIT_TOKEN=$$($(ENGINE_BIN) config init-token --config $(DEV_CONF)); \
		echo "=============================================="; \
		echo "  INIT_TOKEN=$$INIT_TOKEN"; \
		echo "=============================================="; \
	else \
		echo "✓ Config already exists at $(DEV_CONF)"; \
	fi

# First-time setup: deps + UI + build + migrate + config
watch-init: watch-deps watch-ui watch-build
	@$(MAKE) watch-db --no-print-directory
	@$(MAKE) watch-conf --no-print-directory
	@$(MAKE) watch-install-binaries --no-print-directory
	@echo ""
	@echo "✓ Ready! Run 'make watch' to start CDS."

# Build Angular UI for embedding (skips if engine/ui/dist/ already has files)
watch-ui:
	@if [ -d engine/ui/dist ] && [ "$$(find engine/ui/dist -type f ! -name .gitkeep | head -1)" != "" ]; then \
		echo "  UI already built (engine/ui/dist/ has files), skipping. Use 'make watch-ui-rebuild' to force."; \
	else \
		echo "▶ Building Angular UI (first time, this takes a few minutes)..."; \
		(cd ui && npm ci --prefer-offline && \
		node --max-old-space-size=3048 node_modules/@angular/cli/bin/ng build ui-ng2 --configuration production && \
		node --max-old-space-size=3048 node_modules/@angular/cli/bin/ng build workflow-graph --configuration production) && \
		echo "▶ Embedding UI files..." && \
		rm -rf engine/ui/dist && mkdir -p engine/ui/dist && \
		cp -r ui/dist/browser/* engine/ui/dist/ && \
		echo "  UI files embedded successfully"; \
	fi

# Force rebuild Angular UI
watch-ui-rebuild:
	@echo "▶ Rebuilding Angular UI..."
	@(cd ui && npm ci --prefer-offline && \
	node --max-old-space-size=3048 node_modules/@angular/cli/bin/ng build ui-ng2 --configuration production && \
	node --max-old-space-size=3048 node_modules/@angular/cli/bin/ng build workflow-graph --configuration production)
	@echo "▶ Embedding UI files..."
	@rm -rf engine/ui/dist && mkdir -p engine/ui/dist && cp -r ui/dist/browser/* engine/ui/dist/
	@echo "  UI files embedded successfully"

# Build engine, worker and cdsctl using go build directly (always detects source changes)
watch-build:
	@echo "▶ Building engine..."
	@mkdir -p engine/dist engine/worker/dist cli/cdsctl/dist
	@cd engine && CGO_ENABLED=0 go build $(DEV_LDFLAGS) -o dist/cds-engine-$(DEV_GOOS)-$(DEV_GOARCH) .
	@echo "▶ Building worker..."
	@cd engine/worker && CGO_ENABLED=0 go build -ldflags "\
		-X github.com/ovh/cds/sdk.VERSION=$(DEV_VERSION) \
		-X github.com/ovh/cds/sdk.GOOS=$(DEV_GOOS) \
		-X github.com/ovh/cds/sdk.GOARCH=$(DEV_GOARCH) \
		-X github.com/ovh/cds/sdk.GITHASH=$(DEV_GITHASH) \
		-X github.com/ovh/cds/sdk.BINARY=cds-worker" \
		-o dist/cds-worker-$(DEV_GOOS)-$(DEV_GOARCH) .
	@echo "▶ Building cdsctl..."
	@cd cli/cdsctl && CGO_ENABLED=0 go build $(DEV_LDFLAGS) -o dist/cdsctl-$(DEV_GOOS)-$(DEV_GOARCH) .

# Copy worker/engine/cdsctl binaries into the download directory so the API finds them
watch-install-binaries:
	@mkdir -p $(DEV_DIR)
	@for bin in $(WORKER_BIN) $(ENGINE_BIN) $(CDSCTL_BIN); do \
		if [ -f "$$bin" ]; then \
			cp -f "$$bin" $(DEV_DIR)/; \
			echo "  copied $$(basename $$bin) → $(DEV_DIR)/"; \
		fi; \
	done

# Default admin credentials for make watch
DEV_ADMIN_USER ?= $(shell whoami)
DEV_ADMIN_PASS ?= admin

# Build and run all services in a single process
watch: watch-ui watch-build watch-install-binaries watch-db
	@echo "▶ Creating admin user (idempotent)..."
	@$(ENGINE_BIN) database create-admin \
		--config $(DEV_CONF) \
		--username $(DEV_ADMIN_USER) --password $(DEV_ADMIN_PASS) \
		--email $(DEV_ADMIN_USER)@localhost.local
	@ln -sf cdsctl-$(DEV_GOOS)-$(DEV_GOARCH) cli/cdsctl/dist/cdsctl
	@echo "▶ Auto-login cdsctl + import plugins + setup forgejo in background..."
	@( while ! $(CDSCTL_BIN) login --api-url http://localhost:8081/cdsapi --driver local \
		--username $(DEV_ADMIN_USER) --password $(DEV_ADMIN_PASS) \
		--no-interactive --context dev 2>/dev/null; do \
		sleep 2; \
	done; \
	echo "✓ cdsctl logged in"; \
	PATH="$(CURDIR)/cli/cdsctl/dist:$$PATH" $(MAKE) watch-plugins --no-print-directory || \
		echo "✗ watch-plugins failed"; \
	PATH="$(CURDIR)/cli/cdsctl/dist:$$PATH" $(MAKE) watch-setup-forgejo --no-print-directory || \
		echo "✗ watch-setup-forgejo failed" ) &
	@echo "▶ Starting CDS (all services)..."
	@$(ENGINE_BIN) start api cdn ui hooks repositories vcs elasticsearch hatchery:local --config $(DEV_CONF)

# Build and publish v2 action plugins. Can also be run standalone when CDS is running.
watch-plugins:
	@echo "▶ Creating cdsctl symlink..."
	@ln -sf cdsctl-$(DEV_GOOS)-$(DEV_GOARCH) cli/cdsctl/dist/cdsctl
	@echo "▶ Building v2 action plugins ($(DEV_GOOS)/$(DEV_GOARCH) only)..."
	@PATH="$(CURDIR)/cli/cdsctl/dist:$(PATH)" $(MAKE) build -C contrib/grpcplugins/action OS="$(DEV_GOOS)" ARCH="$(DEV_GOARCH)"
	@echo "▶ Publishing v2 action plugins..."
	@PATH="$(CURDIR)/cli/cdsctl/dist:$(PATH)" $(MAKE) publish -C contrib/grpcplugins/action OS="$(DEV_GOOS)" ARCH="$(DEV_GOARCH)"
	@echo "✓ All v2 plugins published"

# Create a Forgejo API token, a CDS project, and wire the VCS connection.
# Idempotent: skips steps that are already done.
watch-setup-forgejo:
	@echo "▶ Setting up Forgejo VCS connection..."
	@FORGEJO_TOKEN=$$(curl -sf \
		-X POST \
		-u $(DEV_ADMIN_USER):$(DEV_ADMIN_PASS) \
		-H "Content-Type: application/json" \
		-d '{"name":"cds-dev","scopes":["all"]}' \
		http://localhost:$(DEV_FORGEJO_PORT)/api/v1/users/$(DEV_ADMIN_USER)/tokens \
		| grep -o '"sha1":"[^"]*"' | cut -d'"' -f4); \
	if [ -z "$$FORGEJO_TOKEN" ]; then \
		echo "  token 'cds-dev' may already exist, fetching..."; \
		FORGEJO_TOKEN=$$(curl -sf \
			-u $(DEV_ADMIN_USER):$(DEV_ADMIN_PASS) \
			-X DELETE \
			http://localhost:$(DEV_FORGEJO_PORT)/api/v1/users/$(DEV_ADMIN_USER)/tokens/cds-dev && \
		curl -sf \
			-X POST \
			-u $(DEV_ADMIN_USER):$(DEV_ADMIN_PASS) \
			-H "Content-Type: application/json" \
			-d '{"name":"cds-dev","scopes":["all"]}' \
			http://localhost:$(DEV_FORGEJO_PORT)/api/v1/users/$(DEV_ADMIN_USER)/tokens \
			| grep -o '"sha1":"[^"]*"' | cut -d'"' -f4); \
	fi; \
	if [ -z "$$FORGEJO_TOKEN" ]; then \
		echo "✗ Failed to get Forgejo API token"; exit 1; \
	fi; \
	echo "  Forgejo API token OK"; \
	echo "  creating CDS project DEMO (idempotent)..."; \
	cdsctl project create DEMO "Demo Project" 2>/dev/null || true; \
	echo "  adding Forgejo VCS to project DEMO..."; \
	TMP=$$(mktemp); \
	printf 'name: forgejo\ntype: gitea\nurl: "http://localhost:$(DEV_FORGEJO_PORT)"\nauth:\n  username: $(DEV_ADMIN_USER)\n  token: "%s"\n' "$$FORGEJO_TOKEN" > "$$TMP"; \
	cdsctl project vcs import DEMO "$$TMP" --force 2>/dev/null \
		&& echo "✓ Forgejo VCS configured on project DEMO" \
		|| echo "  Forgejo VCS already configured or import failed"; \
	rm -f "$$TMP"
