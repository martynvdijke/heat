bool _asBool(dynamic value) =>
    value is bool ? value : value is num && value != 0;

class RoundScore {
  final int id;
  final int snapshotId;
  final int racerId;
  final String racerName;
  final int points;
  final int position;
  final bool dnf;
  final bool dns;
  final int spins;
  final bool overheated;

  const RoundScore({
    required this.id,
    required this.snapshotId,
    required this.racerId,
    required this.racerName,
    required this.points,
    required this.position,
    required this.dnf,
    required this.dns,
    required this.spins,
    required this.overheated,
  });

  factory RoundScore.fromJson(Map<String, dynamic> j) => RoundScore(
    id: (j['id'] as num).toInt(),
    snapshotId: (j['snapshot_id'] as num).toInt(),
    racerId: (j['racer_id'] as num).toInt(),
    racerName: j['racer_name'] as String,
    points: (j['points'] as num).toInt(),
    position: (j['position'] as num).toInt(),
    dnf: _asBool(j['dnf']),
    dns: _asBool(j['dns']),
    spins: (j['spins'] as num?)?.toInt() ?? 0,
    overheated: _asBool(j['overheated']),
  );
}

class RoundSnapshot {
  final int id;
  final int seasonId;
  final String raceName;
  final String? raceDate;
  final int round;
  final String? createdAt;
  final String status; // draft|final
  final List<RoundScore> scores;

  const RoundSnapshot({
    required this.id,
    required this.seasonId,
    required this.raceName,
    this.raceDate,
    required this.round,
    this.createdAt,
    required this.status,
    required this.scores,
  });

  factory RoundSnapshot.fromJson(Map<String, dynamic> j) => RoundSnapshot(
    id: (j['id'] as num).toInt(),
    seasonId: (j['season_id'] as num).toInt(),
    raceName: j['race_name'] as String,
    raceDate: j['race_date'] as String?,
    round: (j['round'] as num).toInt(),
    createdAt: j['created_at'] as String?,
    status: j['status'] as String? ?? 'draft',
    scores: ((j['scores'] as List?) ?? [])
        .map((e) => RoundScore.fromJson(e as Map<String, dynamic>))
        .toList(),
  );
}
