import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:heat/models/upgrade.dart';
import 'package:heat/providers/app_providers.dart';
import 'package:heat/screens/upgrades_screen.dart';
import 'package:heat/theme/app_theme.dart';

UpgradesSnapshot _snapshot({int seasonId = 3}) => UpgradesSnapshot(
  seasonId: seasonId,
  upgrades: MyUpgrades(
    owned: [
      PlayerUpgrade(
        id: 11,
        racerId: 7,
        upgradeId: 3,
        seasonId: 3,
        equipped: true,
        roundBought: 2,
        upgrade: const UpgradeCard(
          id: 3,
          name: 'Turbo',
          description: 'Extra speed down the straight.',
          cardType: 'upgrade',
          cost: 12,
          effects: '{}',
          extensionId: 0,
        ),
      ),
    ],
    available: const [
      UpgradeCard(
        id: 5,
        name: 'Brakes',
        description: 'Shorter pit stops.',
        cardType: 'upgrade',
        cost: 8,
        effects: '{}',
        extensionId: 0,
      ),
    ],
  ),
);

Future<void> _pump(WidgetTester tester, UpgradesSnapshot snapshot) async {
  await tester.pumpWidget(
    ProviderScope(
      overrides: [myUpgradesProvider.overrideWith((ref) async => snapshot)],
      child: MaterialApp(theme: AppTheme.dark, home: const UpgradesScreen()),
    ),
  );
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('shows owned and available upgrade cards', (tester) async {
    await _pump(tester, _snapshot());

    expect(find.byKey(const Key('upgradesScreen')), findsOneWidget);
    expect(find.text('Turbo'), findsOneWidget);
    expect(find.text('Brakes'), findsOneWidget);
    expect(find.text('Unequip'), findsOneWidget);
    expect(find.byKey(const Key('toggleUpgrade_11')), findsOneWidget);
    expect(find.byKey(const Key('buyUpgrade_5')), findsOneWidget);
  });

  testWidgets('disables buying when there is no active season', (tester) async {
    await _pump(tester, _snapshot(seasonId: 0));

    final buy = tester.widget<ButtonStyleButton>(
      find.byKey(const Key('buyUpgrade_5')),
    );
    expect(buy.onPressed, isNull);
    expect(find.textContaining('No active season'), findsOneWidget);
  });

  testWidgets('shows empty states', (tester) async {
    await _pump(
      tester,
      const UpgradesSnapshot(
        seasonId: 3,
        upgrades: MyUpgrades(owned: [], available: []),
      ),
    );

    expect(find.text('No upgrades owned yet.'), findsOneWidget);
    expect(find.text('No upgrades available to buy.'), findsOneWidget);
  });
}
