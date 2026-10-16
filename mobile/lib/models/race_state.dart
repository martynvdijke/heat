class RaceState {
  final String state; // stopped|racing|paused
  final int elapsedMs;
  final int currentLap;
  final int totalLaps;

  const RaceState({
    required this.state,
    required this.elapsedMs,
    required this.currentLap,
    required this.totalLaps,
  });

  factory RaceState.fromJson(Map<String, dynamic> j) => RaceState(
        state: j['state'] as String,
        elapsedMs: (j['elapsed_ms'] as num).toInt(),
        currentLap: (j['current_lap'] as num).toInt(),
        totalLaps: (j['total_laps'] as num).toInt(),
      );

  Map<String, dynamic> toJson() => {
        'state': state,
        'elapsed_ms': elapsedMs,
        'current_lap': currentLap,
        'total_laps': totalLaps,
      };
}
