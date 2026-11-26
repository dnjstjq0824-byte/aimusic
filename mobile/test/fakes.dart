// Shared test fakes: scriptable Dio transport + in-memory secure storage.
library;

import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_secure_storage/test/test_flutter_secure_storage_platform.dart';
import 'package:flutter_secure_storage_platform_interface/flutter_secure_storage_platform_interface.dart';

/// Installs in-memory secure storage, returning the backing map.
Map<String, String> mockSecureStorage([Map<String, String>? seed]) {
  final data = <String, String>{...?seed};
  FlutterSecureStoragePlatform.instance =
      TestFlutterSecureStoragePlatform(data);
  return data;
}

typedef AdapterHandler = Future<ResponseBody> Function(
    RequestOptions options);

/// Scriptable Dio transport: records requests, replays canned responses.
class FakeAdapter implements HttpClientAdapter {
  FakeAdapter(this._handler);

  final AdapterHandler _handler;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) {
    requests.add(options);
    return _handler(options);
  }

  @override
  void close({bool force = false}) {}
}

/// JSON response body for [FakeAdapter] scripts.
ResponseBody jsonResponse(Object? body, int status) =>
    ResponseBody.fromString(
      jsonEncode(body),
      status,
      headers: {Headers.contentTypeHeader: [Headers.jsonContentType]},
    );
