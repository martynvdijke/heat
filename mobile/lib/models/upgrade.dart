/// Upgrade cards owned by a racer, and cards available to buy.
///
/// Mirrors `models.UpgradeCard` / `models.PlayerUpgrade` on the server.
class UpgradeCard {
  final int id;
  final String name;
  final String description;
  final String cardType;
  final int cost;
  final String effects;
  final int extensionId;

  const UpgradeCard({
    required this.id,
    required this.name,
    required this.description,
    required this.cardType,
    required this.cost,
    required this.effects,
    required this.extensionId,
  });

  factory UpgradeCard.fromJson(Map<String, dynamic> j) => UpgradeCard(
    id: (j['id'] as num?)?.toInt() ?? 0,
    name: j['name'] as String? ?? '',
    description: j['description'] as String? ?? '',
    cardType: j['card_type'] as String? ?? '',
    cost: (j['cost'] as num?)?.toInt() ?? 0,
    effects: j['effects'] as String? ?? '',
    extensionId: (j['extension_id'] as num?)?.toInt() ?? 0,
  );
}

class PlayerUpgrade {
  final int id;
  final int racerId;
  final int upgradeId;
  final int seasonId;
  final bool equipped;
  final int roundBought;
  final UpgradeCard? upgrade;

  const PlayerUpgrade({
    required this.id,
    required this.racerId,
    required this.upgradeId,
    required this.seasonId,
    required this.equipped,
    required this.roundBought,
    required this.upgrade,
  });

  factory PlayerUpgrade.fromJson(Map<String, dynamic> j) {
    final raw = j['upgrade'];
    return PlayerUpgrade(
      id: (j['id'] as num?)?.toInt() ?? 0,
      racerId: (j['racer_id'] as num?)?.toInt() ?? 0,
      upgradeId: (j['upgrade_id'] as num?)?.toInt() ?? 0,
      seasonId: (j['season_id'] as num?)?.toInt() ?? 0,
      equipped: _asBool(j['equipped']),
      roundBought: (j['round_bought'] as num?)?.toInt() ?? 0,
      upgrade: raw is Map<String, dynamic> ? UpgradeCard.fromJson(raw) : null,
    );
  }

  static bool _asBool(dynamic v) {
    if (v is bool) return v;
    if (v is num) return v != 0;
    return false;
  }
}

class MyUpgrades {
  final List<PlayerUpgrade> owned;
  final List<UpgradeCard> available;

  const MyUpgrades({required this.owned, required this.available});

  factory MyUpgrades.fromJson(Map<String, dynamic> j) => MyUpgrades(
    owned: (j['owned'] as List? ?? const [])
        .whereType<Map<String, dynamic>>()
        .map(PlayerUpgrade.fromJson)
        .toList(),
    available: (j['available'] as List? ?? const [])
        .whereType<Map<String, dynamic>>()
        .map(UpgradeCard.fromJson)
        .toList(),
  );
}
