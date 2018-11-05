// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('grpc');
var actionplugin_pb = require('./actionplugin_pb.js');
var google_protobuf_empty_pb = require('google-protobuf/google/protobuf/empty_pb.js');

function serialize_actionplugin_ActionPluginManifest(arg) {
  if (!(arg instanceof actionplugin_pb.ActionPluginManifest)) {
    throw new Error('Expected argument of type actionplugin.ActionPluginManifest');
  }
  return new Buffer(arg.serializeBinary());
}

function deserialize_actionplugin_ActionPluginManifest(buffer_arg) {
  return actionplugin_pb.ActionPluginManifest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_actionplugin_ActionQuery(arg) {
  if (!(arg instanceof actionplugin_pb.ActionQuery)) {
    throw new Error('Expected argument of type actionplugin.ActionQuery');
  }
  return new Buffer(arg.serializeBinary());
}

function deserialize_actionplugin_ActionQuery(buffer_arg) {
  return actionplugin_pb.ActionQuery.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_actionplugin_ActionResult(arg) {
  if (!(arg instanceof actionplugin_pb.ActionResult)) {
    throw new Error('Expected argument of type actionplugin.ActionResult');
  }
  return new Buffer(arg.serializeBinary());
}

function deserialize_actionplugin_ActionResult(buffer_arg) {
  return actionplugin_pb.ActionResult.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_actionplugin_WorkerHTTPPortQuery(arg) {
  if (!(arg instanceof actionplugin_pb.WorkerHTTPPortQuery)) {
    throw new Error('Expected argument of type actionplugin.WorkerHTTPPortQuery');
  }
  return new Buffer(arg.serializeBinary());
}

function deserialize_actionplugin_WorkerHTTPPortQuery(buffer_arg) {
  return actionplugin_pb.WorkerHTTPPortQuery.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_google_protobuf_Empty(arg) {
  if (!(arg instanceof google_protobuf_empty_pb.Empty)) {
    throw new Error('Expected argument of type google.protobuf.Empty');
  }
  return new Buffer(arg.serializeBinary());
}

function deserialize_google_protobuf_Empty(buffer_arg) {
  return google_protobuf_empty_pb.Empty.deserializeBinary(new Uint8Array(buffer_arg));
}


var ActionPluginService = exports.ActionPluginService = {
  manifest: {
    path: '/actionplugin.ActionPlugin/Manifest',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: actionplugin_pb.ActionPluginManifest,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_actionplugin_ActionPluginManifest,
    responseDeserialize: deserialize_actionplugin_ActionPluginManifest,
  },
  run: {
    path: '/actionplugin.ActionPlugin/Run',
    requestStream: false,
    responseStream: false,
    requestType: actionplugin_pb.ActionQuery,
    responseType: actionplugin_pb.ActionResult,
    requestSerialize: serialize_actionplugin_ActionQuery,
    requestDeserialize: deserialize_actionplugin_ActionQuery,
    responseSerialize: serialize_actionplugin_ActionResult,
    responseDeserialize: deserialize_actionplugin_ActionResult,
  },
  workerHTTPPort: {
    path: '/actionplugin.ActionPlugin/WorkerHTTPPort',
    requestStream: false,
    responseStream: false,
    requestType: actionplugin_pb.WorkerHTTPPortQuery,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_actionplugin_WorkerHTTPPortQuery,
    requestDeserialize: deserialize_actionplugin_WorkerHTTPPortQuery,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
  stop: {
    path: '/actionplugin.ActionPlugin/Stop',
    requestStream: false,
    responseStream: false,
    requestType: google_protobuf_empty_pb.Empty,
    responseType: google_protobuf_empty_pb.Empty,
    requestSerialize: serialize_google_protobuf_Empty,
    requestDeserialize: deserialize_google_protobuf_Empty,
    responseSerialize: serialize_google_protobuf_Empty,
    responseDeserialize: deserialize_google_protobuf_Empty,
  },
};

exports.ActionPluginClient = grpc.makeGenericClientConstructor(ActionPluginService);
