DIST := dist

.PHONY: clean package

$(DIST):
	mkdir -p $(DIST)

workflow-graph:
	cd ../../ui && npm ci && ng build workflow-graph

package-workflow-graph:
    cd ../../ui/dist/workflow-graph && npm pack

webviews: package-workflow-graph
	cd webviews/workflow-preview && npm install && npm run build

build: $(DIST) webviews
	npm install --include=dev
	npm run vsce package
	mv vscode-cds-*.vsix $(DIST)

package: $(TARGET_DIST) build
	cp -f $(DIST)/vscode-cds-*.vsix $(TARGET_DIST)

clean:
	rm -f $(DIST)/vscode-cds-*.vsix
