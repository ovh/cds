.PHONY: clean package

VERSION := $(if ${CDS_VERSION},${CDS_VERSION},0.0.0-snapshot)
TARGET_DIST := ./dist

$(TARGET_DIST):
	@mkdir -p $(TARGET_DIST)

workflow-graph:
	cd ../../ui && npm ci && npm run ng -- build workflow-graph

package-workflow-graph: workflow-graph
	cd ../../ui/dist/workflow-graph && npm pack

webviews: package-workflow-graph
	cd webviews/workflow-preview && npm install && npm run build

plugin: $(TARGET_DIST)
	npm ci
	npm run vsce -- package $(VERSION) --no-update-package-json -o $(TARGET_DIST)/vscode-cds.vsix

build: webviews plugin

clean:
	rm -rf ./dist-web ./out $(TARGET_DIST)

install:
	code --install-extension $(TARGET_DIST)/vscode-cds.vsix --force