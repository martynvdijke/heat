import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/upgrade.dart';
import '../providers/app_providers.dart';
import '../providers/session_providers.dart';
import '../theme/app_theme.dart';

/// Upgrade cards owned by the racer plus the cards still available to buy for
/// the current season. Mirrors the upgrade section on the web `/me` page.
class UpgradesScreen extends ConsumerStatefulWidget {
  const UpgradesScreen({super.key});

  @override
  ConsumerState<UpgradesScreen> createState() => _UpgradesScreenState();
}

class _UpgradesScreenState extends ConsumerState<UpgradesScreen> {
  final Set<int> _busyToggles = <int>{};
  final Set<int> _busyBuys = <int>{};

  String _message(Object error) {
    final text = error.toString();
    const prefix = 'Exception: ';
    return text.startsWith(prefix) ? text.substring(prefix.length) : text;
  }

  void _snack(String message) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
  }

  Future<void> _refresh() async {
    ref.invalidate(myUpgradesProvider);
    try {
      await ref.read(myUpgradesProvider.future);
    } catch (_) {
      // The provider renders its own error state.
    }
  }

  Future<void> _toggle(PlayerUpgrade pu) async {
    setState(() => _busyToggles.add(pu.id));
    try {
      await ref.read(apiClientProvider).toggleUpgrade(id: pu.id, equipped: !pu.equipped);
      ref.invalidate(myUpgradesProvider);
    } catch (e) {
      if (mounted) _snack(_message(e));
    } finally {
      if (mounted) setState(() => _busyToggles.remove(pu.id));
    }
  }

  Future<void> _buy(UpgradeCard card, int seasonId) async {
    setState(() => _busyBuys.add(card.id));
    try {
      await ref.read(apiClientProvider).buyUpgrade(upgradeId: card.id, seasonId: seasonId);
      ref.invalidate(myUpgradesProvider);
    } catch (e) {
      if (mounted) _snack(_message(e));
    } finally {
      if (mounted) setState(() => _busyBuys.remove(card.id));
    }
  }

  @override
  Widget build(BuildContext context) {
    final upgradesAsync = ref.watch(myUpgradesProvider);
    return Scaffold(
      key: const Key('upgradesScreen'),
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        title: const Text('UPGRADE CARDS'),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(1),
          child: Container(height: 1, color: AppTheme.borderCol),
        ),
      ),
      body: upgradesAsync.when(
        loading: () => const Center(child: CircularProgressIndicator(color: AppTheme.accent)),
        error: (e, _) => _ErrorState(
          message: _message(e),
          onRetry: () => ref.invalidate(myUpgradesProvider),
        ),
        data: (snapshot) => _buildContent(snapshot),
      ),
    );
  }

  Widget _buildContent(UpgradesSnapshot snapshot) {
    final owned = snapshot.upgrades.owned;
    final available = snapshot.upgrades.available;
    final equipped = owned.where((u) => u.equipped).length;

    return RefreshIndicator(
      color: AppTheme.accent,
      backgroundColor: AppTheme.surface2,
      onRefresh: _refresh,
      child: ListView(
        physics: const AlwaysScrollableScrollPhysics(),
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 28),
        children: [
          _SummaryPanel(
            ownedCount: owned.length,
            equippedCount: equipped,
            seasonId: snapshot.seasonId,
          ),
          if (snapshot.seasonId == 0) ...[
            const SizedBox(height: 10),
            const _SeasonNotice(),
          ],
          const SizedBox(height: 22),
          _SectionLabel(label: 'OWNED', trailing: '${owned.length}'),
          const SizedBox(height: 10),
          if (owned.isEmpty)
            const _EmptyPanel(
              icon: Icons.style_outlined,
              text: 'No upgrades owned yet.',
            )
          else
            ...owned.map(
              (pu) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: _ownedCard(pu),
              ),
            ),
          const SizedBox(height: 22),
          _SectionLabel(label: 'AVAILABLE', trailing: '${available.length}'),
          const SizedBox(height: 10),
          if (available.isEmpty)
            const _EmptyPanel(
              icon: Icons.shopping_bag_outlined,
              text: 'No upgrades available to buy.',
            )
          else
            ...available.map(
              (card) => Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: _availableCard(card, snapshot.seasonId),
              ),
            ),
        ],
      ),
    );
  }

  Widget _ownedCard(PlayerUpgrade pu) {
    final card = pu.upgrade;
    final name = (card != null && card.name.trim().isNotEmpty) ? card.name : 'Upgrade #${pu.upgradeId}';
    final description = card?.description.trim() ?? '';
    final cost = card?.cost ?? 0;
    final busy = _busyToggles.contains(pu.id);
    final tint = pu.equipped ? AppTheme.stateRacing : AppTheme.accent;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppTheme.surface2,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(
          color: pu.equipped
              ? AppTheme.stateRacing.withValues(alpha: 0.30)
              : AppTheme.borderCol,
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _IconTile(icon: Icons.style, color: tint),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: AppTheme.textPrimary,
                      ),
                    ),
                    if (description.isNotEmpty) ...[
                      const SizedBox(height: 3),
                      Text(
                        description,
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 12,
                          height: 1.4,
                          color: AppTheme.textSecondary,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: Wrap(
                  spacing: 8,
                  runSpacing: 6,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    _CostChip(cost: cost),
                    if (pu.equipped) const _EquippedBadge(),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              _toggleButton(pu, busy),
            ],
          ),
        ],
      ),
    );
  }

  Widget _availableCard(UpgradeCard card, int seasonId) {
    final busy = _busyBuys.contains(card.id);
    final buyingDisabled = seasonId == 0 || busy;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppTheme.surface2,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: AppTheme.borderCol),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _IconTile(icon: Icons.shopping_bag_outlined, color: AppTheme.accent),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      card.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w700,
                        color: AppTheme.textPrimary,
                      ),
                    ),
                    if (card.description.trim().isNotEmpty) ...[
                      const SizedBox(height: 3),
                      Text(
                        card.description.trim(),
                        maxLines: 3,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 12,
                          height: 1.4,
                          color: AppTheme.textSecondary,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              _CostChip(cost: card.cost),
              const Spacer(),
              ElevatedButton(
                key: Key('buyUpgrade_${card.id}'),
                onPressed: buyingDisabled ? null : () => _buy(card, seasonId),
                style: ElevatedButton.styleFrom(
                  minimumSize: const Size(88, 36),
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                ),
                child: busy ? const _ButtonSpinner() : const Text('Buy'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _toggleButton(PlayerUpgrade pu, bool busy) {
    if (pu.equipped) {
      return OutlinedButton(
        key: Key('toggleUpgrade_${pu.id}'),
        onPressed: busy ? null : () => _toggle(pu),
        style: OutlinedButton.styleFrom(
          minimumSize: const Size(96, 36),
          padding: const EdgeInsets.symmetric(horizontal: 14),
        ),
        child: busy ? const _ButtonSpinner() : const Text('Unequip'),
      );
    }
    return ElevatedButton(
      key: Key('toggleUpgrade_${pu.id}'),
      onPressed: busy ? null : () => _toggle(pu),
      style: ElevatedButton.styleFrom(
        minimumSize: const Size(96, 36),
        padding: const EdgeInsets.symmetric(horizontal: 16),
      ),
      child: busy ? const _ButtonSpinner() : const Text('Equip'),
    );
  }
}

class _SummaryPanel extends StatelessWidget {
  final int ownedCount;
  final int equippedCount;
  final int seasonId;

  const _SummaryPanel({
    required this.ownedCount,
    required this.equippedCount,
    required this.seasonId,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(14, 12, 14, 14),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppTheme.surface2, AppTheme.surface],
        ),
        borderRadius: BorderRadius.circular(AppTheme.radiusLg),
        border: Border.all(color: AppTheme.borderCol),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.build_outlined, size: 13, color: AppTheme.accent),
              const SizedBox(width: 7),
              const Text(
                'GARAGE',
                style: TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 1.8,
                  color: AppTheme.textDim,
                ),
              ),
              const Spacer(),
              if (seasonId > 0)
                Text(
                  'SEASON $seasonId',
                  style: const TextStyle(
                    fontSize: 9,
                    fontWeight: FontWeight.w800,
                    letterSpacing: 1.2,
                    color: AppTheme.textDim,
                    fontFeatures: [FontFeature.tabularFigures()],
                  ),
                ),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(child: _Stat(value: '$ownedCount', label: 'OWNED')),
              Container(width: 1, height: 30, color: AppTheme.borderCol),
              Expanded(
                child: Padding(
                  padding: const EdgeInsets.only(left: 14),
                  child: _Stat(
                    value: '$equippedCount',
                    label: 'EQUIPPED',
                    color: equippedCount > 0 ? AppTheme.stateRacing : AppTheme.textPrimary,
                  ),
                ),
              ),
              if (seasonId > 0) ...[
                Container(width: 1, height: 30, color: AppTheme.borderCol),
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.only(left: 14),
                    child: _Stat(value: '#$seasonId', label: 'SEASON'),
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }
}

class _Stat extends StatelessWidget {
  final String value;
  final String label;
  final Color color;

  const _Stat({
    required this.value,
    required this.label,
    this.color = AppTheme.textPrimary,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          value,
          style: TextStyle(
            fontSize: 20,
            fontWeight: FontWeight.w900,
            letterSpacing: -0.3,
            color: color,
            fontFeatures: const [FontFeature.tabularFigures()],
          ),
        ),
        const SizedBox(height: 2),
        Text(
          label,
          style: const TextStyle(
            fontSize: 9,
            fontWeight: FontWeight.w800,
            letterSpacing: 1.2,
            color: AppTheme.textDim,
          ),
        ),
      ],
    );
  }
}

class _SectionLabel extends StatelessWidget {
  final String label;
  final String? trailing;

  const _SectionLabel({required this.label, this.trailing});

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 3,
          height: 16,
          decoration: BoxDecoration(
            color: AppTheme.accent,
            borderRadius: BorderRadius.circular(99),
          ),
        ),
        const SizedBox(width: 8),
        Text(
          label,
          style: const TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w800,
            letterSpacing: 1.4,
            color: AppTheme.textSecondary,
          ),
        ),
        const Spacer(),
        if (trailing != null)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
            decoration: BoxDecoration(
              color: AppTheme.surface2,
              borderRadius: BorderRadius.circular(99),
              border: Border.all(color: AppTheme.borderCol),
            ),
            child: Text(
              trailing!,
              style: const TextStyle(
                fontSize: 9,
                fontWeight: FontWeight.w700,
                letterSpacing: 0.8,
                color: AppTheme.textDim,
                fontFeatures: [FontFeature.tabularFigures()],
              ),
            ),
          ),
      ],
    );
  }
}

class _IconTile extends StatelessWidget {
  final IconData icon;
  final Color color;

  const _IconTile({required this.icon, required this.color});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 38,
      height: 38,
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: color.withValues(alpha: 0.25)),
      ),
      child: Icon(icon, size: 18, color: color),
    );
  }
}

class _CostChip extends StatelessWidget {
  final int cost;

  const _CostChip({required this.cost});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: AppTheme.surface3,
        borderRadius: BorderRadius.circular(99),
        border: Border.all(color: AppTheme.borderCol),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.toll, size: 12, color: AppTheme.accent),
          const SizedBox(width: 5),
          Text(
            '$cost pts',
            style: const TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w700,
              color: AppTheme.textSecondary,
              fontFeatures: [FontFeature.tabularFigures()],
            ),
          ),
        ],
      ),
    );
  }
}

class _EquippedBadge extends StatelessWidget {
  const _EquippedBadge();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 4),
      decoration: BoxDecoration(
        color: AppTheme.stateRacing.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(99),
        border: Border.all(color: AppTheme.stateRacing.withValues(alpha: 0.25)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 5,
            height: 5,
            decoration: const BoxDecoration(
              color: AppTheme.stateRacing,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 5),
          const Text(
            'EQUIPPED',
            style: TextStyle(
              fontSize: 9,
              fontWeight: FontWeight.w800,
              letterSpacing: 0.8,
              color: AppTheme.stateRacing,
            ),
          ),
        ],
      ),
    );
  }
}

class _SeasonNotice extends StatelessWidget {
  const _SeasonNotice();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: AppTheme.accent.withValues(alpha: 0.10),
        borderRadius: BorderRadius.circular(AppTheme.radiusSm),
        border: Border.all(color: AppTheme.accent.withValues(alpha: 0.30)),
      ),
      child: Row(
        children: [
          const Icon(Icons.info_outline, size: 15, color: AppTheme.accent),
          const SizedBox(width: 8),
          const Expanded(
            child: Text(
              'No active season. Buying is unavailable.',
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                height: 1.35,
                color: AppTheme.accent,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _EmptyPanel extends StatelessWidget {
  final IconData icon;
  final String text;

  const _EmptyPanel({required this.icon, required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 22),
      decoration: BoxDecoration(
        color: AppTheme.surface,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: AppTheme.borderCol),
      ),
      child: Column(
        children: [
          Icon(icon, size: 20, color: AppTheme.textDim),
          const SizedBox(height: 8),
          Text(
            text,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 12, color: AppTheme.textDim),
          ),
        ],
      ),
    );
  }
}

class _ButtonSpinner extends StatelessWidget {
  const _ButtonSpinner();

  @override
  Widget build(BuildContext context) {
    return const SizedBox(
      width: 14,
      height: 14,
      child: CircularProgressIndicator(
        strokeWidth: 2,
        color: AppTheme.textSecondary,
      ),
    );
  }
}

class _ErrorState extends StatelessWidget {
  final String message;
  final VoidCallback onRetry;

  const _ErrorState({required this.message, required this.onRetry});

  @override
  Widget build(BuildContext context) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            color: AppTheme.surface2,
            borderRadius: BorderRadius.circular(AppTheme.radiusLg),
            border: Border.all(color: AppTheme.borderCol),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 42,
                height: 42,
                decoration: BoxDecoration(
                  color: AppTheme.accent2.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(AppTheme.radiusSm),
                  border: Border.all(color: AppTheme.accent2.withValues(alpha: 0.25)),
                ),
                child: const Icon(Icons.wifi_off, size: 20, color: AppTheme.accent2),
              ),
              const SizedBox(height: 14),
              const Text(
                'Could not load upgrade cards',
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w700,
                  color: AppTheme.textPrimary,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                message,
                textAlign: TextAlign.center,
                maxLines: 6,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 12,
                  height: 1.4,
                  color: AppTheme.textSecondary,
                ),
              ),
              const SizedBox(height: 16),
              ElevatedButton.icon(
                key: const Key('retryUpgrades'),
                onPressed: onRetry,
                icon: const Icon(Icons.refresh, size: 16),
                label: const Text('Retry'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
