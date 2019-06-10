reset:
	rm -rf node_modules
	rm -rf semantic
	rm -rf semantic\\
	rm package-lock.json

install:
	npm install

stats:
	node --max_old_space_size=4096 node_modules/@angular/cli/bin/ng build --prod --stats-json
