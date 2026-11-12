// F-07: loopback matcher for the release API guard.
import 'package:avibe/core/env.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('loopback hosts are detected', () {
    expect(Env.isLoopbackUrl('http://localhost:8080'), isTrue);
    expect(Env.isLoopbackUrl('http://localhost/api/v1/tracks'), isTrue);
    expect(Env.isLoopbackUrl('http://127.0.0.1:8080'), isTrue);
    expect(Env.isLoopbackUrl('http://[::1]:8080'), isTrue);
    expect(Env.isLoopbackUrl('http://10.0.2.2:8080'), isTrue);
    expect(Env.isLoopbackUrl('HTTP://LOCALHOST:8080'), isTrue);
  });

  test('public hosts are not loopback', () {
    expect(Env.isLoopbackUrl('https://api.example.com'), isFalse);
    expect(Env.isLoopbackUrl('https://api.example.com/api/v1/tracks'), isFalse);
    expect(Env.isLoopbackUrl('http://192.168.1.10:8080'), isFalse);
    expect(Env.isLoopbackUrl('not a url'), isFalse);
    expect(Env.isLoopbackUrl(''), isFalse);
  });
}
