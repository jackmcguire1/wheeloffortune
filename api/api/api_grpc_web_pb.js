/**
 * @fileoverview gRPC-Web generated client stub for gopher
 * @enhanceable
 * @public
 */

// GENERATED CODE -- DO NOT EDIT!


/* eslint-disable */
// @ts-nocheck



const grpc = {};
grpc.web = require('grpc-web');

const proto = {};
proto.gopher = require('./api_pb.js');

/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?grpc.web.ClientOptions} options
 * @constructor
 * @struct
 * @final
 */
proto.gopher.WheelClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options.format = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @param {string} hostname
 * @param {?Object} credentials
 * @param {?grpc.web.ClientOptions} options
 * @constructor
 * @struct
 * @final
 */
proto.gopher.WheelPromiseClient =
    function(hostname, credentials, options) {
  if (!options) options = {};
  options.format = 'text';

  /**
   * @private @const {!grpc.web.GrpcWebClientBase} The client
   */
  this.client_ = new grpc.web.GrpcWebClientBase(options);

  /**
   * @private @const {string} The hostname
   */
  this.hostname_ = hostname;

};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopher.CreateWheelReq,
 *   !proto.gopher.CreateWheelResp>}
 */
const methodDescriptor_Wheel_CreateWheel = new grpc.web.MethodDescriptor(
  '/gopher.Wheel/CreateWheel',
  grpc.web.MethodType.UNARY,
  proto.gopher.CreateWheelReq,
  proto.gopher.CreateWheelResp,
  /**
   * @param {!proto.gopher.CreateWheelReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopher.CreateWheelResp.deserializeBinary
);


/**
 * @param {!proto.gopher.CreateWheelReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.RpcError, ?proto.gopher.CreateWheelResp)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopher.CreateWheelResp>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopher.WheelClient.prototype.createWheel =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopher.Wheel/CreateWheel',
      request,
      metadata || {},
      methodDescriptor_Wheel_CreateWheel,
      callback);
};


/**
 * @param {!proto.gopher.CreateWheelReq} request The
 *     request proto
 * @param {?Object<string, string>=} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopher.CreateWheelResp>}
 *     Promise that resolves to the response
 */
proto.gopher.WheelPromiseClient.prototype.createWheel =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopher.Wheel/CreateWheel',
      request,
      metadata || {},
      methodDescriptor_Wheel_CreateWheel);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopher.SpinWheelReq,
 *   !proto.gopher.SpinWheelResp>}
 */
const methodDescriptor_Wheel_SpinWheel = new grpc.web.MethodDescriptor(
  '/gopher.Wheel/SpinWheel',
  grpc.web.MethodType.UNARY,
  proto.gopher.SpinWheelReq,
  proto.gopher.SpinWheelResp,
  /**
   * @param {!proto.gopher.SpinWheelReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopher.SpinWheelResp.deserializeBinary
);


/**
 * @param {!proto.gopher.SpinWheelReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.RpcError, ?proto.gopher.SpinWheelResp)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopher.SpinWheelResp>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopher.WheelClient.prototype.spinWheel =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopher.Wheel/SpinWheel',
      request,
      metadata || {},
      methodDescriptor_Wheel_SpinWheel,
      callback);
};


/**
 * @param {!proto.gopher.SpinWheelReq} request The
 *     request proto
 * @param {?Object<string, string>=} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopher.SpinWheelResp>}
 *     Promise that resolves to the response
 */
proto.gopher.WheelPromiseClient.prototype.spinWheel =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopher.Wheel/SpinWheel',
      request,
      metadata || {},
      methodDescriptor_Wheel_SpinWheel);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopher.WheelStatusReq,
 *   !proto.gopher.WheelStatusResp>}
 */
const methodDescriptor_Wheel_GetWheelStatus = new grpc.web.MethodDescriptor(
  '/gopher.Wheel/GetWheelStatus',
  grpc.web.MethodType.UNARY,
  proto.gopher.WheelStatusReq,
  proto.gopher.WheelStatusResp,
  /**
   * @param {!proto.gopher.WheelStatusReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopher.WheelStatusResp.deserializeBinary
);


/**
 * @param {!proto.gopher.WheelStatusReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.RpcError, ?proto.gopher.WheelStatusResp)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopher.WheelStatusResp>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopher.WheelClient.prototype.getWheelStatus =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopher.Wheel/GetWheelStatus',
      request,
      metadata || {},
      methodDescriptor_Wheel_GetWheelStatus,
      callback);
};


/**
 * @param {!proto.gopher.WheelStatusReq} request The
 *     request proto
 * @param {?Object<string, string>=} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopher.WheelStatusResp>}
 *     Promise that resolves to the response
 */
proto.gopher.WheelPromiseClient.prototype.getWheelStatus =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopher.Wheel/GetWheelStatus',
      request,
      metadata || {},
      methodDescriptor_Wheel_GetWheelStatus);
};


/**
 * @const
 * @type {!grpc.web.MethodDescriptor<
 *   !proto.gopher.GetAllWheelnamesReq,
 *   !proto.gopher.GetAllWheelnamesResp>}
 */
const methodDescriptor_Wheel_GetAllWheelNames = new grpc.web.MethodDescriptor(
  '/gopher.Wheel/GetAllWheelNames',
  grpc.web.MethodType.UNARY,
  proto.gopher.GetAllWheelnamesReq,
  proto.gopher.GetAllWheelnamesResp,
  /**
   * @param {!proto.gopher.GetAllWheelnamesReq} request
   * @return {!Uint8Array}
   */
  function(request) {
    return request.serializeBinary();
  },
  proto.gopher.GetAllWheelnamesResp.deserializeBinary
);


/**
 * @param {!proto.gopher.GetAllWheelnamesReq} request The
 *     request proto
 * @param {?Object<string, string>} metadata User defined
 *     call metadata
 * @param {function(?grpc.web.RpcError, ?proto.gopher.GetAllWheelnamesResp)}
 *     callback The callback function(error, response)
 * @return {!grpc.web.ClientReadableStream<!proto.gopher.GetAllWheelnamesResp>|undefined}
 *     The XHR Node Readable Stream
 */
proto.gopher.WheelClient.prototype.getAllWheelNames =
    function(request, metadata, callback) {
  return this.client_.rpcCall(this.hostname_ +
      '/gopher.Wheel/GetAllWheelNames',
      request,
      metadata || {},
      methodDescriptor_Wheel_GetAllWheelNames,
      callback);
};


/**
 * @param {!proto.gopher.GetAllWheelnamesReq} request The
 *     request proto
 * @param {?Object<string, string>=} metadata User defined
 *     call metadata
 * @return {!Promise<!proto.gopher.GetAllWheelnamesResp>}
 *     Promise that resolves to the response
 */
proto.gopher.WheelPromiseClient.prototype.getAllWheelNames =
    function(request, metadata) {
  return this.client_.unaryCall(this.hostname_ +
      '/gopher.Wheel/GetAllWheelNames',
      request,
      metadata || {},
      methodDescriptor_Wheel_GetAllWheelNames);
};


module.exports = proto.gopher;

