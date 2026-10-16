import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../models/race_state.dart';
import '../models/racer.dart';
import '../models/standing.dart';
import '../models/ws_envelope.dart';

class LiveState {
  final List<Racer> racers;
  final List<Standing> standings;
  final RaceState? raceState;
  final String? flag;
  final int lastSeq;
  const LiveState({this.racers = const [], this.standings = const [], this.raceState, this.flag, this.lastSeq = 0});

  LiveState copyWith({List<Racer>? racers, List<Standing>? standings, RaceState? raceState, String? flag, int? lastSeq}) =>
      LiveState(
        racers: racers ?? this.racers,
        standings: standings ?? this.standings,
        raceState: raceState ?? this.raceState,
        flag: flag ?? this.flag,
        lastSeq: lastSeq ?? this.lastSeq,
      );
}

class LiveNotifier extends StateNotifier<LiveState> {
  LiveNotifier() : super(const LiveState());

  void apply(WsEnvelope env) {
    if (WsEnvelope.hasGap(state.lastSeq, env.seq) && env.seq != 0) {
      // gap handled by WsClient resync; just note
    }
    final seq = env.seq != 0 ? env.seq : state.lastSeq;
    switch (env.type) {
      case 'racers':
        if (env.payload is List) {
          final list = (env.payload as List).map((e) => Racer.fromJson(e as Map<String, dynamic>)).toList();
          state = state.copyWith(racers: list, lastSeq: seq);
        }
        break;
      case 'standings':
        if (env.payload is List) {
          final list = (env.payload as List).map((e) => Standing.fromJson(e as Map<String, dynamic>)).toList();
          state = state.copyWith(standings: list, lastSeq: seq);
        }
        break;
      case 'race_state':
        if (env.payload is Map<String, dynamic>) {
          state = state.copyWith(raceState: RaceState.fromJson(env.payload as Map<String, dynamic>), lastSeq: seq);
        }
        break;
      case 'flag':
        state = state.copyWith(flag: env.payload?.toString(), lastSeq: seq);
        break;
      case 'hello':
      case 'resync':
        if (env.payload is Map<String, dynamic>) {
          final snap = env.payload as Map<String, dynamic>;
          _applySnapshot(snap, seq);
        }
        break;
      default:
        if (env.topic == 'racers' && env.payload is List) {
          final list = (env.payload as List).map((e) => Racer.fromJson(e as Map<String, dynamic>)).toList();
          state = state.copyWith(racers: list, lastSeq: seq);
        } else if (env.topic == 'standings' && env.payload is List) {
          final list = (env.payload as List).map((e) => Standing.fromJson(e as Map<String, dynamic>)).toList();
          state = state.copyWith(standings: list, lastSeq: seq);
        } else if (env.topic == 'race_state' && env.payload is Map<String, dynamic>) {
          state = state.copyWith(raceState: RaceState.fromJson(env.payload as Map<String, dynamic>), lastSeq: seq);
        }
    }
  }

  void _applySnapshot(Map<String, dynamic> snap, int seq) {
    List<Racer>? racers;
    List<Standing>? standings;
    RaceState? rs;
    if (snap['racers'] is List) racers = (snap['racers'] as List).map((e) => Racer.fromJson(e as Map<String, dynamic>)).toList();
    if (snap['standings'] is List) standings = (snap['standings'] as List).map((e) => Standing.fromJson(e as Map<String, dynamic>)).toList();
    if (snap['race_state'] is Map<String, dynamic>) rs = RaceState.fromJson(snap['race_state'] as Map<String, dynamic>);
    state = state.copyWith(racers: racers, standings: standings, raceState: rs, lastSeq: seq);
  }
}

final liveProvider = StateNotifierProvider<LiveNotifier, LiveState>((ref) => LiveNotifier());
