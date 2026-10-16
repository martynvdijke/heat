class Season {
  final int id;
  final String name;
  final String? startDate;
  final String? endDate;
  final String status; // active|archived
  final String? createdAt;
  final List<int> moduleIds;

  const Season({
    required this.id,
    required this.name,
    this.startDate,
    this.endDate,
    required this.status,
    this.createdAt,
    required this.moduleIds,
  });

  factory Season.fromJson(Map<String, dynamic> j) => Season(
        id: j['id'] as int,
        name: j['name'] as String,
        startDate: j['start_date'] as String?,
        endDate: j['end_date'] as String?,
        status: j['status'] as String,
        createdAt: j['created_at'] as String?,
        moduleIds: ((j['module_ids'] as List?) ?? []).map((e) => (e as num).toInt()).toList(),
      );
}
