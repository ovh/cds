// launch grpc_tools_node_protoc --js_out=import_style=commonjs,binary:. --grpc_out=. --plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` --proto_path=../../../../../sdk/grpcplugin/actionplugin actionplugin.proto

const messages = require('./actionplugin_pb');
const services = require('./actionplugin_grpc_pb');
const google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
const grpc = require('grpc');
const os = require('os');
const getPort = require('get-port');
let httpPort;

function manifest(call, callback) {
  var reply = new messages.ActionPluginManifest();
  reply.setName('plugin-nodejs');
  callback(null, reply);
}

function run(call, callback) {
  var reply = new messages.ActionResult();
  console.log('This is a test');
  reply.setStatus('Success');
  callback(null, reply);
}

function workerHTTPPort(call, callback) {
  var reply = new google_protobuf_empty_pb.Empty();
  httpPort = call.request;
  callback(null, reply);
}

function main() {
  let localIP = '127.0.0.1';
  let server = new grpc.Server();
  server.addService(services.ActionPluginService, {run, workerHTTPPort, manifest});
  getPort()
    .then((port) => {
      server.bind(localIP + ':' + port, grpc.ServerCredentials.createInsecure());
      server.start();
      console.log(`${localIP}:${port} is ready to accept new connection`);//must be in this mandatory form (worker will use this log));
    });
}

main();
