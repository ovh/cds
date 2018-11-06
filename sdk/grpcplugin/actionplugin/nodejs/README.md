# SDK Node.js for CDS GRPC action plugins

## How to use

+ First you have to install GRPC Tools:

```
$ npm install -g grpc-tools
```

+ You have to create a link on npm locally with this package by typing the following command in its directory:

```
$ npm link
```

+ Then go to the directory of your plugin and launch:

```
$ npm link sdk-grpcplugin-cds-nodejs
```

+ Now you can develop locally with the SDK (example below):

```javascript
const sdk = require('sdk-grpcplugin-cds-nodejs');
const path = require('path');

// Here is the function that is launched when your step is running
function run(call, callback) {
  console.log('This is a test');
  sdk.Client.success('', callback); // There are 2 helpers sdk.Client.success and sdk.Client.fail to return the right status and the message linked
}

function main() {
  let client = new sdk.Client(path.join(__dirname, './nodejs.yml'), run); //Indicate the yaml file which describe your plugin
  client.start();
}

main();
```

You can find a richer example [here](https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/examples/nodejs)
