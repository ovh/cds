DIST := dist

.PHONY: clean package

$(DIST):
	mkdir -p $(DIST)

build: $(DIST)
	cd ../../ui && npm ci && ng build workflow-graph && cd dist/workflow-graph && npm pack
	npm install --include=dev
	cd webviews/workflow-preview && npm install && npm run build
	npm run vsce package
	mv vscode-cds-*.vsix $(DIST)

package: $(TARGET_DIST) build
	cp -f $(DIST)/vscode-cds-*.vsix $(TARGET_DIST)

clean:
	rm -f $(DIST)/vscode-cds-*.vsix
