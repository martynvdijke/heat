import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'session_providers.dart';
import '../models/racer.dart';
import '../models/racer_stats.dart';
import '../models/round_snapshot.dart';
import '../models/season.dart';
import '../models/upgrade.dart';

/// Owned + available upgrade cards together with the current season id used
/// when buying a card.
class UpgradesSnapshot {
  final int seasonId;
  final MyUpgrades upgrades;
  const UpgradesSnapshot({required this.seasonId, required this.upgrades});
}

final myUpgradesProvider = FutureProvider<UpgradesSnapshot>((ref) async {
  final api = ref.watch(apiClientProvider);
  final meFuture = api.fetchMe();
  final upgradesFuture = api.fetchMyUpgrades();
  final me = await meFuture;
  final upgrades = await upgradesFuture;
  final season = me['season'];
  final seasonId = season is Map ? (season['id'] as num?)?.toInt() ?? 0 : 0;
  return UpgradesSnapshot(seasonId: seasonId, upgrades: upgrades);
});

final racersProvider = FutureProvider<List<Racer>>(
  (ref) => ref.watch(apiClientProvider).fetchRacers(),
);
final seasonsProvider = FutureProvider<List<Season>>(
  (ref) => ref.watch(apiClientProvider).fetchSeasons(),
);
final racerStatsListProvider = FutureProvider<List<RacerStats>>(
  (ref) => ref.watch(apiClientProvider).fetchRacerStatsList(),
);
final seasonRoundsProvider = FutureProvider.family<List<RoundSnapshot>, int>(
  (ref, seasonId) =>
      ref.watch(apiClientProvider).fetchRoundsForSeason(seasonId),
);
