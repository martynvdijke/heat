import 'package:flutter_riverpod/legacy.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../services/session_store.dart';
import '../services/api_client.dart';

final sharedPrefsProvider = FutureProvider<SharedPreferences>((ref) async => SharedPreferences.getInstance());

final sessionStoreProvider = Provider<SessionStore>((ref) {
  throw UnimplementedError('Override in app.dart with real prefs');
});

final apiClientProvider = Provider<ApiClient>((ref) {
  final s = ref.watch(sessionStoreProvider);
  return ApiClient(store: s);
});

final baseUrlProvider = StateProvider<String?>((ref) => ref.watch(sessionStoreProvider).baseUrl);
