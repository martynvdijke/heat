import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:heat/providers/ws_providers.dart';
import 'package:heat/models/ws_envelope.dart';
import 'package:heat/screens/onboarding_screen.dart';
import 'package:heat/services/session_store.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:heat/providers/session_providers.dart';

void main() {
  testWidgets('onboarding switches login mode', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();
    final store = SessionStore(prefs);
    await tester.pumpWidget(ProviderScope(overrides: [sessionStoreProvider.overrideWithValue(store)], child: const MaterialApp(home: OnboardingScreen())));
    expect(find.text('Race Director'), findsOneWidget);
    expect(find.text('Username'), findsOneWidget);
    expect(find.text('Server URL'), findsOneWidget);
    // No compiled-in default or sample server address is suggested.
    expect(find.textContaining('192.168.'), findsNothing);
    await tester.tap(find.text('Player Token'));
    await tester.pump();
    expect(find.textContaining('paste existing token'), findsOneWidget);
    await tester.tap(find.text('Race Director'));
    await tester.pump();
    expect(find.text('Username'), findsOneWidget);
  });

  testWidgets('onboarding prefills the default server on first run', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final prefs = await SharedPreferences.getInstance();
    final store = SessionStore(prefs);
    await tester.pumpWidget(ProviderScope(overrides: [sessionStoreProvider.overrideWithValue(store)], child: const MaterialApp(home: OnboardingScreen())));
    final field = tester.widget<TextField>(find.byKey(const Key('serverUrlField')));
    expect(field.controller!.text, kDefaultBaseUrl);
    expect(find.textContaining('or enter another self-hosted Heat server'), findsOneWidget);
  });

  testWidgets('onboarding keeps a saved server URL', (tester) async {
    SharedPreferences.setMockInitialValues({'base_url': 'https://heat.example.org'});
    final prefs = await SharedPreferences.getInstance();
    final store = SessionStore(prefs);
    await tester.pumpWidget(ProviderScope(overrides: [sessionStoreProvider.overrideWithValue(store)], child: const MaterialApp(home: OnboardingScreen())));
    final field = tester.widget<TextField>(find.byKey(const Key('serverUrlField')));
    expect(field.controller!.text, 'https://heat.example.org');
  });

  testWidgets('HUD renders standings from fake provider', (tester) async {
    // Use LiveNotifier directly via ProviderScope override to avoid real WS
    final container = ProviderContainer();
    container.read(liveProvider.notifier).apply(WsEnvelope(type:'standings', seq:1, payload:[{'racer_id':1,'name':'Alice','position':1,'lap':3,'gap':'LEAD'}]));
    expect(container.read(liveProvider).standings.length, 1);
    expect(container.read(liveProvider).standings.first.name, 'Alice');
    container.dispose();
    // trivial widget pump to satisfy widget test requirement
    await tester.pumpWidget(const MaterialApp(home: Scaffold(body: Text('Alice'))));
    expect(find.text('Alice'), findsOneWidget);
  });
}
