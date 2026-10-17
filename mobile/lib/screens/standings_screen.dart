import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';
import '../theme/app_theme.dart';
import '../providers/app_providers.dart';
import '../providers/session_providers.dart';
import '../models/racer_stats.dart';

class StandingsScreen extends ConsumerWidget {
  const StandingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final seasonsAsync = ref.watch(seasonsProvider);
    final statsAsync = ref.watch(racerStatsListProvider);
    final base = ref.watch(sessionStoreProvider).baseUrl ?? '';
    return Scaffold(
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        title: const Text('STANDINGS & STATS'),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 12),
            child: SizedBox(
              height: 34,
              child: ElevatedButton.icon(
                icon: const Icon(Icons.calendar_month, size: 14),
                label: const Text('EXPORT .ICS', style: TextStyle(fontSize: 11, letterSpacing: 0.6)),
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppTheme.surface2,
                  foregroundColor: AppTheme.textPrimary,
                  side: const BorderSide(color: AppTheme.borderStrong),
                  padding: const EdgeInsets.symmetric(horizontal: 12),
                ),
                onPressed: () async {
                  final url = Uri.parse('$base/api/races/export.ics');
                  if (await canLaunchUrl(url)) await launchUrl(url, mode: LaunchMode.externalApplication);
                },
              ),
            ),
          ),
        ],
        bottom: PreferredSize(preferredSize: const Size.fromHeight(1), child: Container(height: 1, color: AppTheme.borderCol)),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          // Seasons
          Row(children: [
            Container(width: 3, height: 16, decoration: BoxDecoration(color: AppTheme.accent, borderRadius: BorderRadius.circular(99))),
            const SizedBox(width: 8),
            const Text('SEASONS', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textSecondary)),
            const Spacer(),
            Container(padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3), decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.borderCol)), child: const Text('TAP FOR ROUNDS', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w700, letterSpacing: 0.8, color: AppTheme.textDim))),
          ]),
          const SizedBox(height: 10),
          seasonsAsync.when(
            loading: () => const _Skeleton(height: 80),
            error: (e, _) => _ErrorBox(msg: '$e'),
            data: (seasons) {
              if (seasons.isEmpty) return const _EmptyBox(text: 'No seasons');
              return Column(
                key: const Key('standingsSeasonList'),
                children: seasons.map((s) {
                  final isActive = s.status == 'active';
                  return Container(
                    margin: const EdgeInsets.only(bottom: 8),
                    decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusMd), border: Border.all(color: isActive ? AppTheme.accent.withValues(alpha: 0.28) : AppTheme.borderCol)),
                    child: Material(
                      color: Colors.transparent,
                      borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                      child: ListTile(
                        contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                        leading: Container(
                          width: 36, height: 36,
                          decoration: BoxDecoration(color: isActive ? AppTheme.accent.withValues(alpha: 0.14) : AppTheme.surface3, borderRadius: BorderRadius.circular(8), border: Border.all(color: isActive ? AppTheme.accent.withValues(alpha: 0.22) : AppTheme.borderCol)),
                          child: Icon(isActive ? Icons.bolt : Icons.flag_outlined, size: 16, color: isActive ? AppTheme.accent : AppTheme.textDim),
                        ),
                        title: Text(s.name, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700, color: AppTheme.textPrimary)),
                        subtitle: Text('${s.status.toUpperCase()}  •  ${s.startDate ?? '—'}  –  ${s.endDate ?? '—'}', style: const TextStyle(fontSize: 11, color: AppTheme.textDim)),
                        trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                          if (isActive) Container(padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3), decoration: BoxDecoration(color: AppTheme.stateRacing.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.stateRacing.withValues(alpha: 0.22))), child: const Text('ACTIVE', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 0.7, color: AppTheme.stateRacing))),
                          const SizedBox(width: 6),
                          const Icon(Icons.chevron_right, size: 18, color: AppTheme.textDim),
                        ]),
                        onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => SeasonRoundsScreen(seasonId: s.id, seasonName: s.name))),
                      ),
                    ),
                  );
                }).toList(),
              );
            },
          ),
          const SizedBox(height: 20),
          // Racer stats chart
          Row(children: [
            Container(width: 3, height: 16, decoration: BoxDecoration(color: const Color(0xFF2EC4B6), borderRadius: BorderRadius.circular(99))),
            const SizedBox(width: 8),
            const Text('RACER STATS', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textSecondary)),
            const Spacer(),
            const Text('TOP 8 BY POINTS', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w700, letterSpacing: 0.8, color: AppTheme.textDim)),
          ]),
          const SizedBox(height: 10),
          statsAsync.when(
            loading: () => const _Skeleton(height: 200),
            error: (e, _) => _ErrorBox(msg: '$e'),
            data: (stats) {
              if (stats.isEmpty) return const _EmptyBox(text: 'No stats available');
              final top = stats.take(8).toList();
              final maxY = (top.map((e) => e.points).reduce((a, b) => a > b ? a : b).toDouble() * 1.15).clamp(10, 9999).toDouble();
              return Column(children: [
                Container(
                  padding: const EdgeInsets.fromLTRB(8, 14, 14, 8),
                  decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusMd), border: Border.all(color: AppTheme.borderCol)),
                  child: SizedBox(
                    height: 200,
                    child: BarChart(BarChartData(
                      maxY: maxY,
                      gridData: FlGridData(show: true, drawVerticalLine: false, horizontalInterval: (maxY/4).ceilToDouble(), getDrawingHorizontalLine: (_) => FlLine(color: AppTheme.borderCol, strokeWidth: 1)),
                      borderData: FlBorderData(show: false),
                      barGroups: top.asMap().entries.map((e) {
                        final isFirst = e.key == 0;
                        return BarChartGroupData(x: e.key, barRods: [
                          BarChartRodData(
                            toY: e.value.points.toDouble(),
                            width: 18,
                            borderRadius: const BorderRadius.vertical(top: Radius.circular(6)),
                            color: isFirst ? AppTheme.accent : const Color(0xFF2EC4B6),
                            backDrawRodData: BackgroundBarChartRodData(show: true, toY: maxY, color: AppTheme.surface3),
                          )
                        ]);
                      }).toList(),
                      titlesData: FlTitlesData(
                        leftTitles: AxisTitles(sideTitles: SideTitles(showTitles: true, reservedSize: 32, getTitlesWidget: (v, meta) => Text(v.toInt().toString(), style: const TextStyle(fontSize: 9, color: AppTheme.textDim, fontFeatures: [FontFeature.tabularFigures()])))),
                        rightTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
                        topTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
                        bottomTitles: AxisTitles(sideTitles: SideTitles(showTitles: true, getTitlesWidget: (v, meta) {
                          final idx = v.toInt();
                          if (idx < top.length) return Padding(padding: const EdgeInsets.only(top: 6), child: Text('#${top[idx].racerId}', style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w700, color: AppTheme.textDim, fontFeatures: [FontFeature.tabularFigures()])));
                          return const Text('');
                        })),
                      ),
                      barTouchData: BarTouchData(enabled: true, touchTooltipData: BarTouchTooltipData(getTooltipColor: (_) => AppTheme.surface3, getTooltipItem: (g, gi, rod, ri) => BarTooltipItem('${top[g.x].points} pts', const TextStyle(color: AppTheme.textPrimary, fontWeight: FontWeight.w700, fontFeatures: [FontFeature.tabularFigures()])))),
                    )),
                  ),
                ),
                const SizedBox(height: 10),
                ...stats.map((s) => Container(
                      margin: const EdgeInsets.only(bottom: 6),
                      decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.borderCol)),
                      child: Material(
                        color: Colors.transparent,
                        borderRadius: BorderRadius.circular(10),
                        child: ListTile(
                          dense: true,
                          contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                          leading: Container(
                            width: 36, height: 36, alignment: Alignment.center,
                            decoration: BoxDecoration(color: AppTheme.surface3, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)),
                            child: Text('${s.racerId}', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w800, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.textPrimary)),
                          ),
                          title: Row(children: [
                            Text('${s.points}', style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w900, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.accent)),
                            const SizedBox(width: 4),
                            const Text('PTS', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.6, color: AppTheme.textDim)),
                            const SizedBox(width: 8),
                            Container(width: 1, height: 12, color: AppTheme.borderCol),
                            const SizedBox(width: 8),
                            Flexible(child: Text('Racer ${s.racerId}', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: AppTheme.textSecondary), overflow: TextOverflow.ellipsis)),
                          ]),
                          subtitle: Padding(
                            padding: const EdgeInsets.only(top: 2),
                            child: Wrap(spacing: 6, runSpacing: 4, children: [
                              statChip('W', s.wins, AppTheme.accent),
                              statChip('G', s.gold, const Color(0xFFFFD60A)),
                              statChip('S', s.silver, const Color(0xFFB0BEC5)),
                              statChip('B', s.bronze, const Color(0xFFCC7A2A)),
                              statChip('FL', s.fastestLaps, const Color(0xFF7C5CFF)),
                              if (s.dnf > 0) statChip('DNF', s.dnf, AppTheme.stateDnf),
                            ]),
                          ),
                          trailing: const Icon(Icons.chevron_right, size: 16, color: AppTheme.textDim),
                          onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => RacerDetailScreen(racerId: s.racerId))),
                        ),
                      ),
                    )),
              ]);
            },
          ),
        ]),
      ),
    );
  }
}

Widget statChip(String label, int v, Color c) => Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(color: c.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(99), border: Border.all(color: c.withValues(alpha: 0.22))),
      child: Text('$label $v', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, fontFeatures: const [FontFeature.tabularFigures()], color: c)),
    );

class _Skeleton extends StatelessWidget {
  final double height;
  const _Skeleton({required this.height});
  @override
  Widget build(BuildContext context) => Container(height: height, decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(12), border: Border.all(color: AppTheme.borderCol)), child: const Center(child: SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2, color: AppTheme.textDim))));
}
class _ErrorBox extends StatelessWidget {
  final String msg;
  const _ErrorBox({required this.msg});
  @override
  Widget build(BuildContext context) => Container(padding: const EdgeInsets.all(12), decoration: BoxDecoration(color: AppTheme.stateDnf.withValues(alpha: 0.08), borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.stateDnf.withValues(alpha: 0.22))), child: Text('Error: $msg', style: const TextStyle(fontSize: 12, color: AppTheme.stateDnf)));
}
class _EmptyBox extends StatelessWidget {
  final String text;
  const _EmptyBox({required this.text});
  @override
  Widget build(BuildContext context) => Container(padding: const EdgeInsets.all(16), decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.borderCol)), child: Center(child: Text(text, style: const TextStyle(color: AppTheme.textDim))));
}

class SeasonRoundsScreen extends ConsumerWidget {
  final int seasonId;
  final String seasonName;
  const SeasonRoundsScreen({super.key, required this.seasonId, required this.seasonName});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final rounds = ref.watch(seasonRoundsProvider(seasonId));
    return Scaffold(
      backgroundColor: AppTheme.bg,
      appBar: AppBar(title: Text(seasonName.toUpperCase(), style: const TextStyle(letterSpacing: 1.0))),
      body: rounds.when(
        loading: () => const Center(child: CircularProgressIndicator(color: AppTheme.accent)),
        error: (e, _) => Center(child: Text('Error: $e', style: const TextStyle(color: AppTheme.stateDnf))),
        data: (list) => ListView.separated(
          padding: const EdgeInsets.all(16),
          itemCount: list.length,
          separatorBuilder: (context, _) => const SizedBox(height: 8),
          itemBuilder: (_, i) {
            final r = list[i];
            return Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.borderCol)),
              child: Row(children: [
                Container(width: 36, height: 36, alignment: Alignment.center, decoration: BoxDecoration(color: AppTheme.surface3, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)), child: Text('${r.round}', style: const TextStyle(fontWeight: FontWeight.w800, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.textPrimary))),
                const SizedBox(width: 12),
                Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Text(r.raceName, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700, color: AppTheme.textPrimary)),
                  const SizedBox(height: 2),
                  Text('${r.status.toUpperCase()}  •  ${r.scores.length} SCORES', style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w600, letterSpacing: 0.6, color: AppTheme.textDim)),
                ])),
              ]),
            );
          },
        ),
      ),
    );
  }
}

class RacerDetailScreen extends ConsumerWidget {
  final int racerId;
  const RacerDetailScreen({super.key, required this.racerId});
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return FutureBuilder(
      future: ref.read(apiClientProvider).fetchRacerStatsById(racerId),
      builder: (context, snap) {
        if (!snap.hasData) return Scaffold(backgroundColor: AppTheme.bg, appBar: AppBar(title: Text('RACER $racerId')), body: const Center(child: CircularProgressIndicator(color: AppTheme.accent)));
        final data = snap.data!;
        final stats = data['stats'] as Map<String, dynamic>?;
        if (stats == null) return Scaffold(backgroundColor: AppTheme.bg, appBar: AppBar(title: Text('RACER $racerId')), body: const Center(child: Text('No stats', style: TextStyle(color: AppTheme.textDim))));
        final s = RacerStats.fromJson(stats);
        final other = (s.races - s.wins).clamp(0, 9999);
        return Scaffold(
          backgroundColor: AppTheme.bg,
          appBar: AppBar(title: Text('RACER $racerId')),
          body: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(children: [
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusLg), border: Border.all(color: AppTheme.borderCol)),
                child: SizedBox(
                  height: 200,
                  child: PieChart(PieChartData(
                    sectionsSpace: 2,
                    centerSpaceRadius: 54,
                    sections: [
                      PieChartSectionData(value: s.wins.toDouble(), color: AppTheme.accent, title: 'Wins\n${s.wins}', radius: 56, titleStyle: const TextStyle(fontSize: 11, fontWeight: FontWeight.w800, color: Color(0xFF0B0B0B))),
                      PieChartSectionData(value: other.toDouble(), color: AppTheme.surface3, title: 'Other\n$other', radius: 48, titleStyle: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: AppTheme.textDim)),
                    ],
                    pieTouchData: PieTouchData(enabled: true),
                  )),
                ),
              ),
              const SizedBox(height: 14),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.borderCol)),
                child: Row(mainAxisAlignment: MainAxisAlignment.spaceAround, children: [
                  _Kpi(label: 'POINTS', value: '${s.points}', color: AppTheme.accent),
                  Container(width: 1, height: 32, color: AppTheme.borderCol),
                  _Kpi(label: 'RACES', value: '${s.races}', color: AppTheme.textPrimary),
                  Container(width: 1, height: 32, color: AppTheme.borderCol),
                  _Kpi(label: 'WINS', value: '${s.wins}', color: AppTheme.stateRacing),
                ]),
              ),
            ]),
          ),
        );
      },
    );
  }
}

class _Kpi extends StatelessWidget {
  final String label; final String value; final Color color;
  const _Kpi({required this.label, required this.value, required this.color});
  @override
  Widget build(BuildContext context) => Column(children: [
    Text(value, style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900, fontFeatures: const [FontFeature.tabularFigures()], color: color)),
    const SizedBox(height: 2),
    Text(label, style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1.0, color: AppTheme.textDim)),
  ]);
}
