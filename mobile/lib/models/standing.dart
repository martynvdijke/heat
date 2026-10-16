class Standing {
  final int racerId;
  final String name;
  final String? carColor;
  final int position;
  final int lap;
  final String gap; // LEAD / +N / ""

  const Standing({
    required this.racerId,
    required this.name,
    this.carColor,
    required this.position,
    required this.lap,
    required this.gap,
  });

  factory Standing.fromJson(Map<String, dynamic> j) => Standing(
        racerId: (j['racer_id'] as num).toInt(),
        name: j['name'] as String,
        carColor: j['car_color'] as String?,
        position: (j['position'] as num).toInt(),
        lap: (j['lap'] as num).toInt(),
        gap: j['gap'] as String? ?? '',
      );
}
