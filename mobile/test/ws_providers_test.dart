import 'package:flutter_test/flutter_test.dart';
import 'package:heat/models/ws_envelope.dart';
import 'package:heat/providers/ws_providers.dart';

void main() {
  test('hello snapshot populates racers, standings and race state', () {
    final env = WsEnvelope.fromJson({
      'type': 'hello',
      'seq': 5,
      'snapshot': {
        'racers': [
          {'id': 1, 'name': 'Alice', 'points': 10, 'rank': 1, 'position': 1},
        ],
        'standings': [
          {
            'racer_id': 1,
            'name': 'Alice',
            'car_color': '#ff0000',
            'position': 1,
            'lap': 3,
            'gap': 'LEAD',
          },
        ],
        'race_state': {
          'state': 'racing',
          'elapsed_ms': 1000,
          'current_lap': 3,
          'total_laps': 10,
        },
      },
    });

    final notifier = LiveNotifier()..apply(env);

    expect(notifier.state.racers.single.name, 'Alice');
    expect(notifier.state.standings.single.name, 'Alice');
    expect(notifier.state.standings.single.gap, 'LEAD');
    expect(notifier.state.raceState?.currentLap, 3);
    expect(notifier.state.lastSeq, 5);
  });

  test('flag broadcasts use the flag name, not the payload map', () {
    final notifier = LiveNotifier()
      ..apply(
        const WsEnvelope(
          type: 'flag',
          seq: 6,
          payload: {
            'type': 'flag',
            'flag': 'yellow',
            'state': 'on',
            'racer_id': 2,
            'racer_name': 'Bob',
          },
        ),
      );

    expect(notifier.state.flag, 'yellow');
    expect(notifier.state.lastSeq, 6);
  });
}
