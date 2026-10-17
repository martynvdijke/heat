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
    // Launch app with cleared prefs.
    final prefs = await SharedPreferences.getInstance();
    // Ensure clean state after setUp.
    await prefs.clear();

    await app.main();
    await tester.pumpAndSettle(const Duration(seconds: 2));

    // If onboarding is shown, connect to server.
    final serverField = find.byKey(const Key('serverUrlField'));
    if (tester.any(serverField)) {
      await tester.enterText(serverField, _e2eUrl);
      await tester.pumpAndSettle();

      final connectBtn = find.byKey(const Key('connectButton'));
      expect(connectBtn, findsOneWidget);
      await tester.tap(connectBtn);
      await tester.pump();

      // Wait for reachability success (server is up).
      await tester.pumpAndSettle(const Duration(seconds: 10));

      // Verify reachable indicator appeared.
      expect(find.textContaining('Server reachable'), findsOneWidget);
    }

    // Switch to Player login mode.
    final playerMode = find.byKey(const Key('playerLoginMode'));
    if (tester.any(playerMode)) {
      await tester.tap(playerMode);
      await tester.pumpAndSettle();
    } else {
      // Fallback: tap segmented button text if key not hit.
      final alt = find.text('Player Token');
      if (tester.any(alt)) {
        await tester.tap(alt);
        await tester.pumpAndSettle();
      }
    }

    // Wait for racers dropdown to appear after reachability.
    final racerField = find.byKey(const Key('racerField'));
    await tester.pumpAndSettle(const Duration(seconds: 3));

    if (tester.any(racerField)) {
      await tester.tap(racerField);
      await tester.pumpAndSettle();

      // Select racer by name.
      final racerOption = find.text(_e2eRacer).last;
      // Wait up to 5s for dropdown items.
      await _pumpUntilFound(tester, racerOption, timeout: const Duration(seconds: 5));
      expect(racerOption, findsOneWidget);
      await tester.tap(racerOption);
      await tester.pumpAndSettle();
    }

    // Login as selected racer.
    final loginBtn = find.byKey(const Key('loginButton'));
    await _pumpUntilFound(tester, loginBtn, timeout: const Duration(seconds: 5));
    expect(loginBtn, findsOneWidget);
    await tester.tap(loginBtn);
    await tester.pumpAndSettle(const Duration(seconds: 8));

    // Expect HUD leaderboard to appear after navigation to home then HUD.
    // Home screen should be visible after login.
    // Navigate to HUD if not already there.
    final hudDirect = find.byKey(const Key('hudLeaderboard'));
    if (!tester.any(hudDirect)) {
      // Home has nav via HUD card; try to go to HUD.
      // First ensure we are on home, then tap HUD implicitly via navigation?
      // The app navigates to /home after login; HUD is a separate route.
      // Tap the LIVE RACE HUD card text if present.
      final hudCard = find.text('LIVE RACE HUD');
      if (tester.any(hudCard)) {
        await tester.tap(hudCard);
        await tester.pumpAndSettle(const Duration(seconds: 5));
      }
    }

    await _pumpUntilFound(tester, find.byKey(const Key('hudLeaderboard')),
        timeout: const Duration(seconds: 10));
    expect(find.byKey(const Key('hudLeaderboard')), findsOneWidget);

    // Navigate back then to Standings.
    // If we are on HUD, pop.
    if (tester.any(find.text('LIVE RACE'))) {
      await tester.pageBack();
      await tester.pumpAndSettle();
    }

    final navStandings = find.byKey(const Key('navStandings'));
    await _pumpUntilFound(tester, navStandings, timeout: const Duration(seconds: 5));
    expect(navStandings, findsOneWidget);
    await tester.tap(navStandings);
    await tester.pumpAndSettle(const Duration(seconds: 5));

    // Expect season list to contain E2E Season.
    final seasonList = find.byKey(const Key('standingsSeasonList'));
    await _pumpUntilFound(tester, seasonList, timeout: const Duration(seconds: 10));
    expect(seasonList, findsOneWidget);
    expect(find.text('E2E Season'), findsOneWidget);
  });
}

Future<void> _pumpUntilFound(
  WidgetTester tester,
  Finder finder, {
  Duration timeout = const Duration(seconds: 5),
  Duration interval = const Duration(milliseconds: 200),
}) async {
  final deadline = DateTime.now().add(timeout);
  while (DateTime.now().isBefore(deadline)) {
    if (tester.any(finder)) return;
    await tester.pump(interval);
  }
}
