class RacerStats {
  final int id;
  final int racerId;
  final int races;
  final int wins;
  final int gold;
  final int silver;
  final int bronze;
  final int fastestLaps;
  final int points;
  final int dnf;
  final int dns;
  final int spins;
  final int overheated;

  const RacerStats({
    required this.id,
    required this.racerId,
    required this.races,
    required this.wins,
    required this.gold,
    required this.silver,
    required this.bronze,
    required this.fastestLaps,
    required this.points,
    required this.dnf,
    required this.dns,
    required this.spins,
    required this.overheated,
  });

  factory RacerStats.fromJson(Map<String, dynamic> j) => RacerStats(
        id: (j['id'] as num).toInt(),
        racerId: (j['racer_id'] as num).toInt(),
        races: (j['races'] as num).toInt(),
        wins: (j['wins'] as num).toInt(),
        gold: (j['gold'] as num).toInt(),
        silver: (j['silver'] as num).toInt(),
        bronze: (j['bronze'] as num).toInt(),
        fastestLaps: (j['fastest_laps'] as num).toInt(),
        points: (j['points'] as num).toInt(),
        dnf: (j['dnf'] as num).toInt(),
        dns: (j['dns'] as num).toInt(),
        spins: (j['spins'] as num).toInt(),
        overheated: (j['overheated'] as num).toInt(),
      );
}
