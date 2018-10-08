# SDK Node.Js for CDS GRPC action plugins

## How to use

+ First you have to create a link on npm locally with this package, so in this directory you just have to execute:

```
$ npm link
```

+ Then go to the directory of your plugin and launch:

```
$ npm link sdk-grpcplugin-cds-nodejs
```

+ That's all you just have to write:

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

For a more complete example you can [go there](https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/examples/nodejs)
