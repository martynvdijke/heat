import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'theme/app_theme.dart';
import 'services/session_store.dart';
import 'providers/session_providers.dart';
import 'screens/onboarding_screen.dart';
import 'screens/home_screen.dart';
import 'screens/hud_screen.dart';
import 'screens/snapshot_screen.dart';
import 'screens/standings_screen.dart';

class HeatApp extends ConsumerWidget {
  final SharedPreferences prefs;
  const HeatApp({super.key, required this.prefs});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final store = SessionStore(prefs);
    final hasBase = store.baseUrl != null && store.baseUrl!.isNotEmpty;
    final authed = store.isRaceDirector || store.isPlayer;
    return ProviderScope(
      overrides: [sessionStoreProvider.overrideWithValue(store)],
      child: MaterialApp(
        title: 'Heat',
        theme: AppTheme.dark,
        debugShowCheckedModeBanner: false,
        initialRoute: (hasBase && authed) ? '/home' : '/onboarding',
        routes: {
          '/onboarding': (_) => const OnboardingScreen(),
          '/home': (_) => const HomeScreen(),
          '/hud': (_) => const HudScreen(),
          '/snapshot': (_) => const SnapshotScreen(),
          '/standings': (_) => const StandingsScreen(),
        },
      ),
    );
  }
}
