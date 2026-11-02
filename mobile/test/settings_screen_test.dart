import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:heat/providers/session_providers.dart';
import 'package:heat/screens/settings_screen.dart';
import 'package:heat/services/session_store.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  testWidgets('deletion requires the typed word DELETE', (tester) async {
    SharedPreferences.setMockInitialValues({
      'base_url': 'https://heat.example.org',
      'player_token': 'token',
      'player_racer_id': 7,
      'player_racer_name': 'Alice',
    });
    final prefs = await SharedPreferences.getInstance();
    final store = SessionStore(prefs);
    await tester.pumpWidget(ProviderScope(
      overrides: [sessionStoreProvider.overrideWithValue(store)],
      child: const MaterialApp(home: SettingsScreen()),
    ));

    expect(find.text('Delete racer identity'), findsOneWidget);
    await tester.tap(find.text('Delete racer identity'));
    await tester.pumpAndSettle();

    TextButton confirmButton() =>
        tester.widget<TextButton>(find.byKey(const Key('confirmDeleteButton')));
    expect(confirmButton().onPressed, isNull);

    await tester.enterText(find.byKey(const Key('deleteConfirmField')), 'delete');
    await tester.pump();
    expect(confirmButton().onPressed, isNotNull);

    await tester.enterText(find.byKey(const Key('deleteConfirmField')), 'nope');
    await tester.pump();
    expect(confirmButton().onPressed, isNull);
  });

  testWidgets('settings lists privacy, terms and deletion instructions', (tester) async {
    SharedPreferences.setMockInitialValues({
      'base_url': 'https://heat.example.org',
      'session_cookie': 'abc',
    });
    final prefs = await SharedPreferences.getInstance();
    final store = SessionStore(prefs);
    await tester.pumpWidget(ProviderScope(
      overrides: [sessionStoreProvider.overrideWithValue(store)],
      child: const MaterialApp(home: SettingsScreen()),
    ));

    expect(find.text('Privacy notice'), findsOneWidget);
    expect(find.text('Terms of use'), findsOneWidget);
    expect(find.text('Account deletion instructions'), findsOneWidget);
    expect(find.text('Delete admin account'), findsOneWidget);
  });
}
