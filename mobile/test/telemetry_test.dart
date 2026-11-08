import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:heat/src/telemetry/telemetry.dart';

/// Dio adapter that answers from a canned response without touching a socket.
class _FakeAdapter implements HttpClientAdapter {
  _FakeAdapter({this.statusCode = 200, this.body = '{}', this.throws = false});

  final int statusCode;
  final String body;
  final bool throws;
  final List<RequestOptions> requests = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    if (throws) {
      throw DioException.connectionError(
        requestOptions: options,
        reason: 'offline',
      );
    }
    return ResponseBody.fromString(
      body,
      statusCode,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('normalizeEndpoint', () {
    test('appends /v1/traces', () {
      expect(
        Telemetry.normalizeEndpoint('http://collector:4318'),
        'http://collector:4318/v1/traces',
      );
    });

    test('strips trailing slashes first', () {
      expect(
        Telemetry.normalizeEndpoint('http://collector:4318///'),
        'http://collector:4318/v1/traces',
      );
      expect(
        Telemetry.normalizeEndpoint('  http://collector:4318/  '),
        'http://collector:4318/v1/traces',
      );
    });

    test('preserves an existing /v1/traces suffix', () {
      expect(
        Telemetry.normalizeEndpoint('https://otel.example.com/v1/traces/'),
        'https://otel.example.com/v1/traces',
      );
    });
  });

  group('parseOtelHeaders', () {
    test('parses key=value pairs and decodes values', () {
      final headers = Telemetry.parseOtelHeaders(
        'Authorization=Bearer%20token,x-tenant=heat, broken, =nokey',
      );
      expect(headers, {
        'Authorization': 'Bearer token',
        'x-tenant': 'heat',
      });
    });

    test('empty input yields no headers', () {
      expect(Telemetry.parseOtelHeaders(''), isEmpty);
    });
  });

  group('buildOtlpPayload', () {
    test('resource, scope and span shape', () {
      final span = TraceSpan(
        name: 'GET /api/race/state',
        attributes: const {'http.request.method': 'GET'},
        traceId: 'a' * 32,
        spanId: 'b' * 16,
        startNanos: 1000,
      );

      final payload = buildOtlpPayload(
        serviceName: 'heat-android',
        serviceVersion: '1.2.3',
        span: span,
        attributes: const {
          'http.request.method': 'GET',
          'retry': true,
          'attempt': 2,
          'ratio': 1.5,
        },
        failed: true,
        endNanos: 2000,
      );

      final resourceSpans = payload['resourceSpans'] as List;
      expect(resourceSpans, hasLength(1));
      final resource = resourceSpans.first as Map;
      final resourceAttrs = resource['resource']['attributes'] as List;
      final resourceMap = {
        for (final attr in resourceAttrs)
          attr['key'] as String: attr['value'] as Map,
      };
      expect(resourceMap['service.name'], {'stringValue': 'heat-android'});
      expect(resourceMap['service.version'], {'stringValue': '1.2.3'});
      expect(resourceMap['telemetry.sdk.name'], {'stringValue': 'heat-mobile'});
      expect(resourceMap['telemetry.sdk.language'], {'stringValue': 'dart'});

      final scopeSpans = resource['scopeSpans'] as List;
      final scope = scopeSpans.first as Map;
      expect(scope['scope'], {'name': 'heat.mobile', 'version': '1.2.3'});

      final spans = scope['spans'] as List;
      final exported = spans.first as Map;
      expect(exported['traceId'], 'a' * 32);
      expect(exported['spanId'], 'b' * 16);
      expect(exported['name'], 'GET /api/race/state');
      expect(exported['kind'], 1);
      expect(exported['startTimeUnixNano'], '1000');
      expect(exported['endTimeUnixNano'], '2000');
      expect(exported['status'], {'code': 2});

      final attrs = {
        for (final attr in exported['attributes'] as List)
          attr['key'] as String: attr['value'] as Map,
      };
      expect(attrs['http.request.method'], {'stringValue': 'GET'});
      expect(attrs['retry'], {'boolValue': true});
      expect(attrs['attempt'], {'intValue': '2'});
      expect(attrs['ratio'], {'doubleValue': 1.5});
    });
  });

  group('OtelSettings.fromJson', () {
    test('accepts endpoint/enabled key spelling', () {
      final settings = OtelSettings.fromJson(const {
        'endpoint': 'http://collector:4318',
        'enabled': true,
        'traces_enabled': true,
        'metrics_enabled': false,
        'logs_enabled': true,
      });
      expect(settings.endpoint, 'http://collector:4318');
      expect(settings.enabled, isTrue);
      expect(settings.tracesEnabled, isTrue);
      expect(settings.metricsEnabled, isFalse);
      expect(settings.logsEnabled, isTrue);
    });

    test('accepts otel_endpoint/otel_enabled key spelling', () {
      final settings = OtelSettings.fromJson(const {
        'otel_endpoint': 'https://otel.example.com',
        'otel_enabled': 1,
        'traces_enabled': 'yes',
      });
      expect(settings.endpoint, 'https://otel.example.com');
      expect(settings.enabled, isTrue);
      expect(settings.tracesEnabled, isTrue);
    });

    test('tracesEnabled defaults to enabled', () {
      final settings = OtelSettings.fromJson(const {
        'endpoint': 'http://collector:4318',
        'enabled': true,
      });
      expect(settings.tracesEnabled, isTrue);

      final disabled = OtelSettings.fromJson(const {
        'endpoint': 'http://collector:4318',
        'enabled': false,
      });
      expect(disabled.tracesEnabled, isFalse);
    });

    test('an absent enabled flag means enabled when an endpoint is set', () {
      final settings = OtelSettings.fromJson(const {
        'endpoint': 'http://collector:4318',
        'traces_enabled': true,
      });
      expect(settings.enabled, isTrue);
      expect(settings.tracesEnabled, isTrue);
    });

    test('missing or empty endpoint means disabled', () {
      final missing = OtelSettings.fromJson(const {'enabled': true});
      expect(missing.endpoint, isEmpty);
      expect(missing.enabled, isFalse);

      final blank = OtelSettings.fromJson(const {
        'endpoint': '   ',
        'enabled': true,
        'traces_enabled': true,
      });
      expect(blank.enabled, isFalse);
    });
  });

  group('fetchOtelSettings', () {
    test('returns parsed settings on 200', () async {
      final adapter = _FakeAdapter(
        body: jsonEncode({
          'id': 1,
          'endpoint': 'http://collector:4318',
          'traces_enabled': true,
          'metrics_enabled': false,
          'logs_enabled': false,
        }),
      );
      final dio = Dio()..httpClientAdapter = adapter;

      final settings = await fetchOtelSettings(
        'http://heat.local///',
        dio: dio,
      );

      expect(settings, isNotNull);
      expect(settings!.endpoint, 'http://collector:4318');
      expect(settings.enabled, isTrue);
      expect(settings.tracesEnabled, isTrue);
      expect(adapter.requests.single.uri.toString(),
          'http://heat.local/api/otel-settings');
      expect(adapter.requests.single.headers['Accept'],
          'application/json');
    });

    test('returns null on HTTP error', () async {
      final adapter = _FakeAdapter(statusCode: 401, body: '{"error":"no"}');
      final dio = Dio()..httpClientAdapter = adapter;
      expect(
        await fetchOtelSettings('http://heat.local', dio: dio),
        isNull,
      );
    });

    test('returns null on transport error', () async {
      final adapter = _FakeAdapter(throws: true);
      final dio = Dio()..httpClientAdapter = adapter;
      expect(
        await fetchOtelSettings('http://heat.local', dio: dio),
        isNull,
      );
    });

    test('returns null on malformed JSON', () async {
      final adapter = _FakeAdapter(body: 'not json');
      final dio = Dio()..httpClientAdapter = adapter;
      expect(
        await fetchOtelSettings('http://heat.local', dio: dio),
        isNull,
      );
    });

    test('returns null for an empty base url', () async {
      expect(await fetchOtelSettings('   '), isNull);
    });
  });
}
