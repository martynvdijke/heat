import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'session_providers.dart';
import '../models/racer.dart';
import '../models/racer_stats.dart';
import '../models/round_snapshot.dart';
import '../models/season.dart';

final racersProvider = FutureProvider<List<Racer>>((ref) => ref.watch(apiClientProvider).fetchRacers());
final seasonsProvider = FutureProvider<List<Season>>((ref) => ref.watch(apiClientProvider).fetchSeasons());
final racerStatsListProvider = FutureProvider<List<RacerStats>>((ref) => ref.watch(apiClientProvider).fetchRacerStatsList());
final seasonRoundsProvider = FutureProvider.family<List<RoundSnapshot>, int>((ref, seasonId) => ref.watch(apiClientProvider).fetchRoundsForSeason(seasonId));
