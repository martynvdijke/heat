import 'package:flutter_test/flutter_test.dart';
import 'package:heat/models/racer.dart';
import 'package:heat/models/race_state.dart';
import 'package:heat/models/season.dart';
import 'package:heat/models/round_snapshot.dart';
import 'package:heat/models/standing.dart';
import 'package:heat/models/ws_envelope.dart';
import 'package:heat/models/upgrade.dart';

void main() {
  test('Racer fromJson', () {
    final j = {
      'id': 1,
      'name': 'Alice',
      'points': 10,
      'rank': 1,
      'position': 2,
      'car_color': '#ff0000',
      'car_name': 'Heat',
      'team_id': 3,
      'team_name': 'T',
      'team_color': '#00ff00',
    };
    final r = Racer.fromJson(j);
    expect(r.id, 1);
    expect(r.name, 'Alice');
    expect(r.carColor, '#ff0000');
    expect(r.teamId, 3);
  });
  test('RaceState fromJson', () {
    final rs = RaceState.fromJson({
      'state': 'racing',
      'elapsed_ms': 1234,
      'current_lap': 2,
      'total_laps': 10,
    });
    expect(rs.state, 'racing');
    expect(rs.elapsedMs, 1234);
    expect(rs.totalLaps, 10);
  });
  test('Season fromJson', () {
    final s = Season.fromJson({
      'id': 5,
      'name': 'S1',
      'status': 'active',
      'module_ids': [1, 2],
    });
    expect(s.id, 5);
    expect(s.status, 'active');
    expect(s.moduleIds, [1, 2]);
  });
  test('Standing fromJson', () {
    final st = Standing.fromJson({
      'racer_id': 7,
      'name': 'Bob',
      'position': 1,
      'lap': 3,
      'gap': 'LEAD',
    });
    expect(st.racerId, 7);
    expect(st.gap, 'LEAD');
  });
  test('RoundSnapshot fromJson', () {
    final j = {
      'id': 1,
      'season_id': 2,
      'race_name': 'GP',
      'round': 3,
      'status': 'draft',
      'scores': [
        {
          'id': 10,
          'snapshot_id': 1,
          'racer_id': 1,
          'racer_name': 'A',
          'points': 25,
          'position': 1,
          'dnf': false,
          'dns': false,
          'spins': 0,
          'overheated': false,
        },
      ],
    };
    final snap = RoundSnapshot.fromJson(j);
    expect(snap.id, 1);
    expect(snap.scores.length, 1);
    expect(snap.scores.first.racerName, 'A');
  });
  test('RoundSnapshot fromJson accepts integer overheated flag', () {
    final j = {
      'id': 1,
      'season_id': 2,
      'race_name': 'GP',
      'round': 3,
      'status': 'final',
      'scores': [
        {
          'id': 10,
          'snapshot_id': 1,
          'racer_id': 1,
          'racer_name': 'A',
          'points': 25,
          'position': 1,
          'dnf': 1,
          'dns': 0,
          'spins': 2,
          'overheated': 1,
        },
      ],
    };
    final score = RoundSnapshot.fromJson(j).scores.first;
    expect(score.overheated, isTrue);
    expect(score.dnf, isTrue);
    expect(score.dns, isFalse);
  });
  test('MyUpgrades fromJson parses owned and available cards', () {
    final ups = MyUpgrades.fromJson({
      'owned': [
        {
          'id': 11,
          'racer_id': 7,
          'upgrade_id': 3,
          'season_id': 2,
          'equipped': 1,
          'round_bought': 4,
          'upgrade': {
            'id': 3,
            'name': 'Turbo',
            'description': 'Extra speed',
            'card_type': 'upgrade',
            'cost': 12,
            'effects': '{"speed":1}',
            'extension_id': 0,
          },
        },
      ],
      'available': [
        {
          'id': 5,
          'name': 'Brakes',
          'description': 'Shorter stops',
          'card_type': 'upgrade',
          'cost': 8,
          'effects': '{}',
        },
      ],
    });
    expect(ups.owned.length, 1);
    expect(ups.owned.first.equipped, isTrue);
    expect(ups.owned.first.upgrade?.name, 'Turbo');
    expect(ups.owned.first.roundBought, 4);
    expect(ups.available.single.name, 'Brakes');
    expect(ups.available.single.cost, 8);
  });

  test('MyUpgrades fromJson tolerates missing lists and upgrade', () {
    final ups = MyUpgrades.fromJson({
      'owned': [
        {'id': 1, 'upgrade_id': 2},
      ],
    });
    expect(ups.owned.single.upgrade, isNull);
    expect(ups.owned.single.equipped, isFalse);
    expect(ups.available, isEmpty);
  });

  test('WsEnvelope hasGap', () {
    expect(WsEnvelope.hasGap(5, 7), true);
    expect(WsEnvelope.hasGap(5, 6), false);
    expect(WsEnvelope.hasGap(5, 5), false);
  });
  test('WsEnvelope fromJson', () {
    final e = WsEnvelope.fromJson({
      'type': 'race_state',
      'seq': 42,
      'payload': {
        'state': 'racing',
        'elapsed_ms': 0,
        'current_lap': 1,
        'total_laps': 5,
      },
    });
    expect(e.type, 'race_state');
    expect(e.seq, 42);
    expect(e.payload, isA<Map<String, dynamic>>());
  });
  test('WsEnvelope fromJson reads hello snapshot', () {
    final e = WsEnvelope.fromJson({
      'type': 'hello',
      'seq': 1,
      'snapshot': {
        'standings': [
          {
            'racer_id': 7,
            'name': 'Bob',
            'position': 1,
            'lap': 3,
            'gap': 'LEAD',
          },
        ],
      },
    });
    expect(e.type, 'hello');
    expect(e.payload, isA<Map<String, dynamic>>());
    final snap = e.payload as Map<String, dynamic>;
    expect((snap['standings'] as List).length, 1);
  });
  test('WsEnvelope fromJson prefers snapshot over payload', () {
    final e = WsEnvelope.fromJson({
      'type': 'resync',
      'seq': 2,
      'snapshot': {'racers': []},
      'payload': {
        'racers': [1],
      },
    });
    expect(e.payload, {'racers': []});
  });
}
