// launch grpc_tools_node_protoc --js_out=import_style=commonjs,binary:. --grpc_out=. --plugin=protoc-gen-grpc=`which grpc_tools_node_protoc_plugin` --proto_path=.. actionplugin.proto
const messages = require('./actionplugin_pb');
const services = require('./actionplugin_grpc_pb');
const google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');
const grpc = require('grpc');
const os = require('os');
const getPort = require('get-port');
const YAML = require('yaml')
const fs = require('fs')
let httpPort, pluginManifest, run;

class Client {
  constructor(yamlFilename, runFunc) {
    const file = fs.readFileSync(yamlFilename, 'utf8');
    pluginManifest = YAML.parse(file);
    run = runFunc;
  }

  start(ip) {
    let localIP = ip || '127.0.0.1';
    let server = new grpc.Server();
    server.addService(services.ActionPluginService, {run, workerHTTPPort, manifest});
    return getPort()
      .then((port) => {
        server.bind(localIP + ':' + port, grpc.ServerCredentials.createInsecure());
        server.start();
        console.log(`${localIP}:${port} is ready to accept new connection`);//must be in this mandatory form (worker will use this log));
      });
  }

  static success(msg, callback) {
    let reply = new messages.ActionResult();
    if (msg) {
      reply.setDetails(msg);
    }
    reply.setStatus('Success');
    callback(null, reply);
  }

  static fail(msg, callback) {
    let reply = new messages.ActionResult();
    if (msg) {
      reply.setDetails(msg);
    }
    reply.setStatus('Fail');
    callback(null, reply);
  }
}

function manifest(call, callback) {
  var reply = new messages.ActionPluginManifest();
  reply.setName(pluginManifest.name);
  reply.setDescription(pluginManifest.description);
  reply.setAuthor(pluginManifest.author);
  callback(null, reply);
}

function workerHTTPPort(call, callback) {
  var reply = new google_protobuf_empty_pb.Empty();
  httpPort = call.request;
  callback(null, reply);
}

module.exports = {
  Client
};
