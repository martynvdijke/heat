import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:heat/main.dart' as app;

const String _e2eUrl = String.fromEnvironment('HEAT_E2E_URL',
    defaultValue: 'http://10.0.2.2:6270');
const String _e2eRacer = String.fromEnvironment('HEAT_E2E_RACER',
    defaultValue: 'E2E Racer');

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  setUp(() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();
  });

  testWidgets('E2E: onboarding -> player login -> HUD -> standings',
      (WidgetTester tester) async {
    // Ensure clean state.
    final prefs = await SharedPreferences.getInstance();
    await prefs.clear();

    await app.main();

    // Onboarding screen shows up when no session is stored.
    final serverField = find.byKey(const Key('serverUrlField'));
    await _pumpUntilFound(tester, serverField);
    expect(serverField, findsOneWidget);

    await tester.enterText(serverField, _e2eUrl);
    await _settle(tester);

    final connectBtn = find.byKey(const Key('connectButton'));
    expect(connectBtn, findsOneWidget);
    await tester.tap(connectBtn);

    // Wait for the reachability probe to report success against the live server.
    final reachable = find.textContaining('Server reachable');
    await _pumpUntilFound(tester, reachable, timeout: const Duration(seconds: 20));
    expect(reachable, findsOneWidget);

    // Switch to Player login mode.
    final playerMode = find.byKey(const Key('playerLoginMode'));
    if (tester.any(playerMode)) {
      await tester.tap(playerMode);
      await _settle(tester);
    }

    // Racer dropdown is populated after a successful reachability probe.
    final racerField = find.byKey(const Key('racerField'));
    await _pumpUntilFound(tester, racerField, timeout: const Duration(seconds: 10));
    expect(racerField, findsOneWidget);

    await tester.tap(racerField);
    await _settle(tester);

    final racerOption = find.text(_e2eRacer).last;
    await _pumpUntilFound(tester, racerOption, timeout: const Duration(seconds: 10));
    await tester.tap(racerOption, warnIfMissed: false);
    await _settle(tester);

    // Login as the selected racer.
    final loginBtn = find.byKey(const Key('loginButton'));
    expect(loginBtn, findsOneWidget);
    await tester.tap(loginBtn);

    // Login navigates to the Home screen (has an infinite header pulse, so use
    // bounded pumping rather than pumpAndSettle).
    final homeCard = find.text('LIVE RACE HUD');
    await _pumpUntilFound(tester, homeCard, timeout: const Duration(seconds: 20));
    expect(homeCard, findsOneWidget);

    // Navigate into the live race HUD.
    await tester.ensureVisible(homeCard);
    await _settle(tester, frames: 3);
    await tester.tap(homeCard);
    final hudLeaderboard = find.byKey(const Key('hudLeaderboard'));
    await _pumpUntilFound(tester, hudLeaderboard, timeout: const Duration(seconds: 20));
    expect(hudLeaderboard, findsOneWidget);

    // Back to Home. Wait for the HUD route to actually leave the tree before
    // tapping Home content, otherwise the fading route swallows the hit test.
    await tester.pageBack();
    await _pumpUntilGone(tester, hudLeaderboard, timeout: const Duration(seconds: 10));
    await _settle(tester, frames: 5);

    // Into Standings.
    final navStandings = find.byKey(const Key('navStandings'));
    await _pumpUntilFound(tester, navStandings, timeout: const Duration(seconds: 10));
    expect(navStandings, findsOneWidget);
    await tester.ensureVisible(navStandings);
    await _settle(tester, frames: 3);
    await tester.tap(navStandings);

    final seasonList = find.byKey(const Key('standingsSeasonList'));
    await _pumpUntilFound(tester, seasonList, timeout: const Duration(seconds: 20));
    expect(seasonList, findsOneWidget);

    // The seeded season must be listed.
    final seasonName = find.text('E2E Season');
    await _pumpUntilFound(tester, seasonName, timeout: const Duration(seconds: 10));
    expect(seasonName, findsOneWidget);
  });
}

/// Pumps a fixed number of frames. Safe to use on screens that contain
/// infinite animations, unlike [WidgetTester.pumpAndSettle].
Future<void> _settle(
  WidgetTester tester, {
  int frames = 8,
  Duration step = const Duration(milliseconds: 100),
}) async {
  for (var i = 0; i < frames; i++) {
    await tester.pump(step);
  }
}

/// Pumps frames until [finder] matches at least one widget or [timeout] expires.
Future<bool> _pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 5),
  Duration interval = const Duration(milliseconds: 200),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(deadline)) {
    if (tester.any(finder)) return true;
    await tester.pump(interval);
  }
  return tester.any(finder);
}

/// Pumps frames until [finder] no longer matches any widget or [timeout] expires.
Future<bool> _pumpUntilGone(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 5),
  Duration interval = const Duration(milliseconds: 200),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(deadline)) {
    if (!tester.any(finder)) return true;
    await tester.pump(interval);
  }
  return !tester.any(finder);
}
