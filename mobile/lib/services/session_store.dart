import 'package:shared_preferences/shared_preferences.dart';

class SessionStore {
  static const _kBaseUrl = 'base_url';
  static const _kSessionCookie = 'session_cookie';
  static const _kPlayerToken = 'player_token';
  static const _kPlayerRacerId = 'player_racer_id';
  static const _kPlayerRacerName = 'player_racer_name';

  final SharedPreferences prefs;
  SessionStore(this.prefs);

  String? get baseUrl => prefs.getString(_kBaseUrl);
  String? get sessionCookie => prefs.getString(_kSessionCookie);
  String? get playerToken => prefs.getString(_kPlayerToken);
  int? get playerRacerId => prefs.getInt(_kPlayerRacerId);
  String? get playerRacerName => prefs.getString(_kPlayerRacerName);

  bool get isRaceDirector => sessionCookie != null && sessionCookie!.isNotEmpty;
  bool get isPlayer => playerToken != null && playerToken!.isNotEmpty;

  Future<void> setBaseUrl(String v) => prefs.setString(_kBaseUrl, v);
  Future<void> setSessionCookie(String v) => prefs.setString(_kSessionCookie, v);
  Future<void> setPlayerToken(String token, int racerId, String racerName) async {
    await prefs.setString(_kPlayerToken, token);
    await prefs.setInt(_kPlayerRacerId, racerId);
    await prefs.setString(_kPlayerRacerName, racerName);
  }

  Future<void> clearAuth() async {
    await prefs.remove(_kSessionCookie);
    await prefs.remove(_kPlayerToken);
    await prefs.remove(_kPlayerRacerId);
    await prefs.remove(_kPlayerRacerName);
  }

  String? originForCsrf() {
    final b = baseUrl;
    if (b == null) return null;
    final uri = Uri.tryParse(b);
    if (uri == null) return null;
    return '${uri.scheme}://${uri.host}${uri.hasPort ? ':${uri.port}' : ''}';
  }

  String wsUrl() {
    final b = baseUrl ?? '';
    final uri = Uri.parse(b);
    final scheme = uri.scheme == 'https' ? 'wss' : 'ws';
    final port = uri.hasPort ? ':${uri.port}' : '';
    return '$scheme://${uri.host}$port/ws';
  }
}
