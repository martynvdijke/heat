import 'dart:typed_data';
import 'package:dio/dio.dart';
import 'session_store.dart';
import '../models/racer.dart';
import '../models/race_state.dart';
import '../models/season.dart';
import '../models/round_snapshot.dart';
import '../models/racer_stats.dart';

class ApiClient {
  final Dio dio;
  final SessionStore store;

  ApiClient({required this.store, Dio? dioOverride})
      : dio = dioOverride ?? Dio(BaseOptions(connectTimeout: const Duration(seconds: 5), receiveTimeout: const Duration(seconds: 10)));

  String get _base => store.baseUrl ?? '';

  Map<String, String> _authHeaders({bool mutating = false}) {
    final h = <String, String>{};
    final sc = store.sessionCookie;
    if (sc != null && sc.isNotEmpty) {
      h['Cookie'] = 'session=$sc';
      if (mutating) {
        final o = store.originForCsrf();
        if (o != null) h['Origin'] = o;
      }
    }
    final pt = store.playerToken;
    if (pt != null && pt.isNotEmpty) {
      h['X-Player-Token'] = pt;
    }
    return h;
  }

  String? _extractSessionCookie(Response r) {
    final setCookies = r.headers.map['set-cookie'] ?? r.headers.map['Set-Cookie'] ?? [];
    for (final c in setCookies) {
      final m = RegExp(r'session=([0-9a-fA-F]{64})').firstMatch(c);
      if (m != null) return m.group(1);
    }
    // also try dio headers case-insensitive
    for (final e in r.headers.map.entries) {
      if (e.key.toLowerCase() == 'set-cookie') {
        for (final c in e.value) {
          final m = RegExp(r'session=([0-9a-fA-F]{64})').firstMatch(c);
          if (m != null) return m.group(1);
        }
      }
    }
    return null;
  }

  // Auth
  Future<void> loginRaceDirector(String username, String password) async {
    final res = await dio.post('$_base/api/login',
        data: {'username': username, 'password': password},
        options: Options(headers: mutatingHeaders(), validateStatus: (_) => true));
    if (res.statusCode != 200) {
      throw DioException(requestOptions: res.requestOptions, response: res, message: 'Login failed: ${res.statusCode}');
    }
    final cookie = _extractSessionCookie(res);
    if (cookie == null || cookie.isEmpty) {
      throw Exception('Login succeeded but the server did not return a session cookie');
    }
    await store.setSessionCookie(cookie);
  }

  Map<String, String> mutatingHeaders() {
    final h = <String, String>{};
    final o = store.originForCsrf();
    if (o != null) h['Origin'] = o;
    return h;
  }

  Future<bool> checkOidcEnabled() async {
    try {
      final res = await dio.get('$_base/api/auth/oidc/config');
      if (res.data is Map) return (res.data['enabled'] as bool?) ?? false;
      return false;
    } catch (_) {
      return false;
    }
  }

  /// Starts a native OIDC login. The returned `auth_url` must be opened in an
  /// external browser; the app then polls [pollOIDCMobileLogin] with the
  /// `device_code` until the callback completes.
  Future<Map<String, dynamic>> startOIDCMobileLogin() async {
    final res = await dio.post(
      '$_base/api/auth/oidc/mobile/start',
      options: Options(validateStatus: (_) => true),
    );
    final data = res.data is Map ? Map<String, dynamic>.from(res.data as Map) : <String, dynamic>{};
    if (res.statusCode != 200) {
      throw Exception(data['error'] ?? 'OIDC login unavailable (${res.statusCode})');
    }
    return data;
  }

  /// Polls a native OIDC login. Returns `{status: pending}`, `{status: ok,
  /// session: ...}` or `{status: error, error: ...}`.
  Future<Map<String, dynamic>> pollOIDCMobileLogin(String deviceCode) async {
    final res = await dio.get(
      '$_base/api/auth/oidc/mobile/status',
      queryParameters: {'device': deviceCode},
      options: Options(validateStatus: (_) => true),
    );
    final data = res.data is Map ? Map<String, dynamic>.from(res.data as Map) : <String, dynamic>{};
    if (res.statusCode != 200) {
      throw Exception(data['error'] ?? 'OIDC status failed (${res.statusCode})');
    }
    return data;
  }

  Future<Map<String, dynamic>> loginPlayer(int racerId, String deviceName) async {
    final res = await dio.post('$_base/api/player/login', data: {'racer_id': racerId, 'device_name': deviceName});
    final data = res.data as Map<String, dynamic>;
    final token = data['token'] as String;
    final rid = (data['racer_id'] as num).toInt();
    final rname = data['racer_name'] as String;
    await store.setPlayerToken(token, rid, rname);
    return data;
  }

  Future<Map<String, dynamic>> validatePlayerToken(String token) async {
    final res = await dio.get('$_base/api/player/validate', options: Options(headers: {'X-Player-Token': token}));
    return res.data as Map<String, dynamic>;
  }

  Future<void> logoutPlayer() async {
    try {
      await dio.post('$_base/api/player/logout', options: Options(headers: _authHeaders()));
    } finally {
      await store.clearAuth();
    }
  }

  /// Clears the local session and best-effort invalidates it on the server.
  Future<void> logout() async {
    try {
      if (store.isRaceDirector) {
        await dio.post('$_base/api/logout', options: Options(headers: _authHeaders(mutating: true), validateStatus: (_) => true));
      }
      if (store.isPlayer) {
        await dio.post('$_base/api/player/logout', options: Options(headers: _authHeaders(mutating: true), validateStatus: (_) => true));
      }
    } catch (_) {
      // Best effort; local credentials are always cleared.
    } finally {
      await store.clearAuth();
    }
  }

  // Public
  Future<List<Racer>> fetchRacers() async {
    final res = await dio.get('$_base/api/racers');
    final list = res.data as List;
    return list.map((e) => Racer.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<RaceState> fetchRaceState() async {
    final res = await dio.get('$_base/api/race/state');
    return RaceState.fromJson(res.data as Map<String, dynamic>);
  }

  Future<List<Season>> fetchSeasons() async {
    final res = await dio.get('$_base/api/seasons');
    final list = res.data as List;
    return list.map((e) => Season.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<RacerStats>> fetchRacerStatsList() async {
    final res = await dio.get('$_base/api/racer-stats');
    // server returns list OR wrapper? Handle both
    final data = res.data;
    if (data is List) {
      // Might be list of wrappers {stats:{}, racer:{}}
      // Try to extract stats objects
      final out = <RacerStats>[];
      for (final e in data) {
        if (e is Map<String, dynamic>) {
          if (e.containsKey('stats')) {
            out.add(RacerStats.fromJson(e['stats'] as Map<String, dynamic>));
          } else if (e.containsKey('racer_id')) {
            out.add(RacerStats.fromJson(e));
          }
        }
      }
      return out;
    }
    return [];
  }

  Future<Map<String, dynamic>> fetchRacerStatsById(int id) async {
    final res = await dio.get('$_base/api/racer-stats', queryParameters: {'id': id});
    return res.data as Map<String, dynamic>;
  }

  Future<Uint8List> fetchIcs() async {
    final res = await dio.get('$_base/api/races/export.ics', options: Options(responseType: ResponseType.bytes));
    return Uint8List.fromList(res.data as List<int>);
  }

  // Round snapshot adapted contract
  Future<Map<String, dynamic>> createRoundSnapshot({required String raceName, required int seasonId}) async {
    final res = await dio.post('$_base/api/rounds',
        data: {'race_name': raceName, 'season_id': seasonId, 'round': 0},
        options: Options(headers: _authHeaders(mutating: true)));
    return res.data as Map<String, dynamic>;
  }

  Future<RoundSnapshot> fetchRoundSnapshot(int id) async {
    final res = await dio.get('$_base/api/rounds', queryParameters: {'id': id});
    final data = res.data;
    if (data is Map<String, dynamic>) {
      // Could be { ... snapshot ... } or wrapper
      if (data.containsKey('id') && data.containsKey('scores')) return RoundSnapshot.fromJson(data);
      // If list with single?
      if (data.containsKey('snapshot')) return RoundSnapshot.fromJson(data['snapshot'] as Map<String, dynamic>);
    }
    if (data is List && data.isNotEmpty) {
      return RoundSnapshot.fromJson(data.first as Map<String, dynamic>);
    }
    throw Exception('Unexpected round snapshot response');
  }

  Future<List<RoundSnapshot>> fetchRoundsForSeason(int seasonId) async {
    final res = await dio.get('$_base/api/rounds', queryParameters: {'season_id': seasonId});
    final data = res.data;
    if (data is List) return data.map((e) => RoundSnapshot.fromJson(e as Map<String, dynamic>)).toList();
    if (data is Map<String, dynamic> && data['rounds'] is List) {
      return (data['rounds'] as List).map((e) => RoundSnapshot.fromJson(e as Map<String, dynamic>)).toList();
    }
    return [];
  }

  Future<String?> uploadImage(XFileLike file) async {
    // file abstraction to keep testable; caller passes bytes
    final form = FormData.fromMap({
      'image': MultipartFile.fromBytes(file.bytes, filename: file.name),
    });
    try {
      final res = await dio.post('$_base/api/upload', data: form, options: Options(headers: _authHeaders(mutating: true)));
      final data = res.data as Map<String, dynamic>;
      return data['url'] as String? ?? data['resized_url'] as String? ?? data['thumbnail_url'] as String?;
    } catch (_) {
      return null;
    }
  }

  Future<void> postRaceAction(String action, {int? totalLaps}) async {
    final body = <String, dynamic>{'action': action};
    if (totalLaps != null) body['total_laps'] = totalLaps;
    await dio.post('$_base/api/race/state', data: body, options: Options(headers: _authHeaders(mutating: true)));
  }
}

class XFileLike {
  final List<int> bytes;
  final String name;
  XFileLike(this.bytes, this.name);
}
