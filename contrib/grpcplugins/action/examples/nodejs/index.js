const sdk = require('sdk-grpcplugin-cds-nodejs');
const path = require('path');

function run(call, callback) {
  console.log('This is a test');
  sdk.Client.success('', callback);
}

function main() {
  let client = new sdk.Client(path.join(__dirname, './nodejs.yml'), run);
  client.start();
}

main();
