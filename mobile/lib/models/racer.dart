class Racer {
  final int id;
  final String name;
  final String? profilePicture;
  final String? carColor;
  final String? carName;
  final int points;
  final int rank;
  final int position;
  final int? teamId;
  final String? teamName;
  final String? teamColor;

  const Racer({
    required this.id,
    required this.name,
    this.profilePicture,
    this.carColor,
    this.carName,
    required this.points,
    required this.rank,
    required this.position,
    this.teamId,
    this.teamName,
    this.teamColor,
  });

  factory Racer.fromJson(Map<String, dynamic> j) => Racer(
        id: j['id'] as int,
        name: j['name'] as String,
        profilePicture: j['profile_picture'] as String?,
        carColor: j['car_color'] as String?,
        carName: j['car_name'] as String?,
        points: (j['points'] as num?)?.toInt() ?? 0,
        rank: (j['rank'] as num?)?.toInt() ?? 0,
        position: (j['position'] as num?)?.toInt() ?? 0,
        teamId: j['team_id'] as int?,
        teamName: j['team_name'] as String?,
        teamColor: j['team_color'] as String?,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'profile_picture': profilePicture,
        'car_color': carColor,
        'car_name': carName,
        'points': points,
        'rank': rank,
        'position': position,
        'team_id': teamId,
        'team_name': teamName,
        'team_color': teamColor,
      };
}
