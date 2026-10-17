import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wakelock_plus/wakelock_plus.dart';
import '../theme/app_theme.dart';
import '../providers/ws_providers.dart';
import '../providers/session_providers.dart';
import '../services/ws_client.dart';

class HudScreen extends ConsumerStatefulWidget {
  const HudScreen({super.key});
  @override
  ConsumerState<HudScreen> createState() => _HudScreenState();
}

class _HudScreenState extends ConsumerState<HudScreen> with SingleTickerProviderStateMixin {
  WsClient? _ws;
  late AnimationController _pulse;

  @override
  void initState() {
    super.initState();
    WakelockPlus.enable();
    _pulse = AnimationController(vsync: this, duration: const Duration(milliseconds: 900))..repeat(reverse: true);
    WidgetsBinding.instance.addPostFrameCallback((_) => _connect());
  }

  void _connect() {
    final store = ref.read(sessionStoreProvider);
    final notifier = ref.read(liveProvider.notifier);
    _ws = WsClient(store: store, topics: const ['racers', 'standings', 'race_state', 'flag'], onMessage: notifier.apply);
    _ws!.connect();
  }

  @override
  void dispose() {
    WakelockPlus.disable();
    _pulse.dispose();
    _ws?.dispose();
    super.dispose();
  }

  Future<void> _raceAction(String action) async {
    try {
      await ref.read(apiClientProvider).postRaceAction(action, totalLaps: action == 'start' ? 10 : null);
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Race action failed: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final live = ref.watch(liveProvider);
    final store = ref.watch(sessionStoreProvider);
    final isRD = store.isRaceDirector;
    final myId = store.playerRacerId;
    final rs = live.raceState;
    final standings = live.standings;
    final stateStr = rs?.state ?? 'unknown';
    final isRacing = stateStr == 'racing';

    return Scaffold(
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        title: Row(children: [
          Container(width: 28, height: 28, decoration: BoxDecoration(color: AppTheme.accent, borderRadius: BorderRadius.circular(6)), child: const Icon(Icons.leaderboard, size: 16, color: Color(0xFF0B0B0B))),
          const SizedBox(width: 10),
          const Text('LIVE RACE'),
          const SizedBox(width: 10),
          Container(width: 1, height: 14, color: AppTheme.borderStrong),
          const SizedBox(width: 10),
          Text('${standings.length} RACERS', style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1.2, color: AppTheme.textDim)),
        ]),
        actions: [
          if (isRD) ...[
            _RdBtn(icon: Icons.play_arrow, label: 'Start', onPressed: () => _raceAction('start')),
            _RdBtn(icon: Icons.pause, label: 'Pause', onPressed: () => _raceAction('pause')),
            _RdBtn(icon: Icons.play_circle_outline, label: 'Resume', onPressed: () => _raceAction('resume')),
            _RdBtn(icon: Icons.stop, label: 'Stop', onPressed: () => _raceAction('stop')),
            const SizedBox(width: 4),
          ],
        ],
      ),
      body: SafeArea(
        child: Column(children: [
          // State banner + lap counter strip
          _RaceStateBanner(state: stateStr, flag: live.flag, elapsedMs: rs?.elapsedMs, pulse: _pulse),
          _LapStrip(current: rs?.currentLap, total: rs?.totalLaps, isRacing: isRacing),
          // Column header
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
            color: AppTheme.bgElevated,
            child: Row(children: [
              const SizedBox(width: 36, child: Text('POS', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textDim))),
              const SizedBox(width: 8),
              const Expanded(child: Text('DRIVER', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textDim))),
              SizedBox(width: 36, child: Text('LAP', textAlign: TextAlign.center, style: TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textDim))),
              const SizedBox(width: 8),
              const SizedBox(width: 76, child: Text('GAP', textAlign: TextAlign.right, style: TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textDim))),
            ]),
          ),
          const Divider(height: 1),
          Expanded(
            key: const Key('hudLeaderboard'),
            child: standings.isEmpty
                ? Center(
                    child: Column(mainAxisSize: MainAxisSize.min, children: [
                      Container(
                        width: 56, height: 56,
                        decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(14), border: Border.all(color: AppTheme.borderCol)),
                        child: const Icon(Icons.timer_outlined, color: AppTheme.textDim),
                      ),
                      const SizedBox(height: 12),
                      const Text('Waiting for standings', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w700, color: AppTheme.textSecondary)),
                      const SizedBox(height: 4),
                      const Text('Connected — timing feed will appear here', style: TextStyle(fontSize: 11, color: AppTheme.textDim)),
                    ]),
                  )
                : AnimatedListSwitcher(
                    standings: standings,
                    myId: myId,
                  ),
          ),
        ]),
      ),
    );
  }
}

/// Animated reordering via AnimatedSwitcher keyed rows + implicit motion.
class AnimatedListSwitcher extends StatelessWidget {
  final List<dynamic> standings;
  final int? myId;
  const AnimatedListSwitcher({super.key, required this.standings, required this.myId});

  Color _parseColor(String hex) {
    var h = hex.replaceAll('#', '');
    if (h.length == 6) h = 'FF$h';
    return Color(int.tryParse(h, radix: 16) ?? 0xFFFFFFFF);
  }

  @override
  Widget build(BuildContext context) {
    return ListView.separated(
      itemCount: standings.length,
      separatorBuilder: (context, _) => const Divider(height: 1),
      itemBuilder: (context, i) {
        final s = standings[i];
        final isMe = myId != null && s.racerId == myId;
        final pos = s.position as int;
        final isPodium = pos <= 3;
        return AnimatedContainer(
          duration: const Duration(milliseconds: 320),
          curve: Curves.easeOutCubic,
          decoration: BoxDecoration(
            color: isMe ? AppTheme.accent.withValues(alpha: 0.10) : AppTheme.surface,
            border: isMe ? const Border(left: BorderSide(color: AppTheme.accent, width: 3)) : null,
          ),
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
          child: Row(children: [
            // Pos badge
            Container(
              width: 30, height: 28,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: isPodium ? _podiumBg(pos) : AppTheme.surface2,
                borderRadius: BorderRadius.circular(6),
                border: Border.all(color: isPodium ? _podiumBorder(pos) : AppTheme.borderCol),
              ),
              child: Text('$pos',
                  style: TextStyle(
                    fontFeatures: const [FontFeature.tabularFigures()],
                    fontWeight: FontWeight.w900,
                    fontSize: 14,
                    letterSpacing: -0.3,
                    color: isPodium ? _podiumFg(pos) : AppTheme.textPrimary,
                  )),
            ),
            const SizedBox(width: 10),
            if (s.carColor != null)
              Container(
                width: 10, height: 10,
                margin: const EdgeInsets.only(right: 8),
                decoration: BoxDecoration(
                  color: _parseColor(s.carColor!),
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white24, width: 1),
                  boxShadow: [BoxShadow(color: _parseColor(s.carColor!).withValues(alpha: 0.5), blurRadius: 6)],
                ),
              ),
            Expanded(
              child: Text(s.name as String,
                  maxLines: 1, overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontSize: 13,
                    fontWeight: isMe ? FontWeight.w800 : FontWeight.w600,
                    color: isMe ? AppTheme.textPrimary : AppTheme.textPrimary,
                    letterSpacing: 0.1,
                  )),
            ),
            const SizedBox(width: 8),
            Container(
              width: 36, height: 26,
              alignment: Alignment.center,
              decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(6), border: Border.all(color: AppTheme.borderCol)),
              child: Text('L${s.lap}',
                  style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w800, fontFeatures: [FontFeature.tabularFigures()], letterSpacing: 0.4, color: AppTheme.textSecondary)),
            ),
            const SizedBox(width: 10),
            SizedBox(
              width: 76,
              child: _GapCell(gap: s.gap as String),
            ),
            if (isMe) ...[
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                decoration: BoxDecoration(color: AppTheme.accent, borderRadius: BorderRadius.circular(99)),
                child: const Text('YOU', style: TextStyle(fontSize: 9, fontWeight: FontWeight.w900, letterSpacing: 0.8, color: Color(0xFF0B0B0B))),
              ),
            ],
          ]),
        );
      },
    );
  }

  Color _podiumBg(int pos) {
    switch (pos) {
      case 1: return const Color(0xFFFFD60A).withValues(alpha: 0.18);
      case 2: return const Color(0xFFCFD8DC).withValues(alpha: 0.16);
      case 3: return const Color(0xFFFF8A3D).withValues(alpha: 0.16);
      default: return AppTheme.surface2;
    }
  }
  Color _podiumBorder(int pos) {
    switch (pos) {
      case 1: return const Color(0xFFFFD60A).withValues(alpha: 0.45);
      case 2: return const Color(0xFFB0BEC5).withValues(alpha: 0.45);
      case 3: return const Color(0xFFFF8A3D).withValues(alpha: 0.45);
      default: return AppTheme.borderCol;
    }
  }
  Color _podiumFg(int pos) {
    switch (pos) {
      case 1: return const Color(0xFFFFD60A);
      case 2: return const Color(0xFFE8EEF1);
      case 3: return const Color(0xFFFFAB73);
      default: return AppTheme.textPrimary;
    }
  }
}

class _GapCell extends StatelessWidget {
  final String gap;
  const _GapCell({required this.gap});
  @override
  Widget build(BuildContext context) {
    final isLead = gap.toUpperCase() == 'LEAD';
    final isEmpty = gap.trim().isEmpty;
    if (isEmpty) return const Text('—', textAlign: TextAlign.right, style: TextStyle(color: AppTheme.textDim, fontSize: 12));
    if (isLead) {
      return Container(
        alignment: Alignment.centerRight,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
          decoration: BoxDecoration(color: AppTheme.stateRacing.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.stateRacing.withValues(alpha: 0.28))),
          child: const Text('LEAD', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w900, letterSpacing: 0.8, color: AppTheme.stateRacing)),
        ),
      );
    }
    // normalize +N display tabular
    return Text(gap, textAlign: TextAlign.right, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.textSecondary));
  }
}

class _RaceStateBanner extends StatelessWidget {
  final String state;
  final String? flag;
  final int? elapsedMs;
  final AnimationController pulse;
  const _RaceStateBanner({required this.state, required this.flag, required this.elapsedMs, required this.pulse});
  @override
  Widget build(BuildContext context) {
    final lc = state.toLowerCase();
    final isRacing = lc == 'racing';
    final isPaused = lc == 'paused';
    final bg = AppTheme.stateBg(lc);
    final fg = AppTheme.stateColor(lc);
    return AnimatedContainer(
      duration: const Duration(milliseconds: 300),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: bg,
        border: Border(bottom: BorderSide(color: fg.withValues(alpha: 0.22))),
      ),
      child: Row(children: [
        if (isRacing)
          FadeTransition(
            opacity: Tween(begin: 0.5, end: 1.0).animate(CurvedAnimation(parent: pulse, curve: Curves.easeInOut)),
            child: Container(width: 8, height: 8, decoration: BoxDecoration(color: AppTheme.stateRacing, shape: BoxShape.circle, boxShadow: [BoxShadow(color: AppTheme.stateRacing.withValues(alpha: 0.6), blurRadius: 8)])),
          )
        else
          Container(width: 8, height: 8, decoration: BoxDecoration(color: fg, shape: BoxShape.circle)),
        const SizedBox(width: 10),
        Text(state.toUpperCase(), style: TextStyle(fontSize: 12, fontWeight: FontWeight.w900, letterSpacing: 1.4, color: fg)),
        if (flag != null && flag!.isNotEmpty) ...[
          const SizedBox(width: 10),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.borderCol)),
            child: Row(mainAxisSize: MainAxisSize.min, children: [
              const Icon(Icons.flag, size: 12, color: AppTheme.textSecondary),
              const SizedBox(width: 4),
              Text(flag!, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, color: AppTheme.textSecondary)),
            ]),
          ),
        ],
        const Spacer(),
        if (elapsedMs != null)
          Text(_fmtElapsed(elapsedMs!), style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, fontFeatures: [FontFeature.tabularFigures()], letterSpacing: 0.6, color: AppTheme.textDim)),
        if (isPaused) ...[
          const SizedBox(width: 8),
          Container(width: 1, height: 14, color: fg.withValues(alpha: 0.3)),
          const SizedBox(width: 8),
          Icon(Icons.pause, size: 14, color: fg),
        ],
      ]),
    );
  }
  String _fmtElapsed(int ms) {
    final s = ms ~/ 1000;
    final m = s ~/ 60;
    final sec = s % 60;
    final tenths = (ms % 1000) ~/ 100;
    return '${m.toString().padLeft(2,'0')}:${sec.toString().padLeft(2,'0')}.$tenths';
  }
}

class _LapStrip extends StatelessWidget {
  final int? current;
  final int? total;
  final bool isRacing;
  const _LapStrip({required this.current, required this.total, required this.isRacing});
  @override
  Widget build(BuildContext context) {
    final lapText = (current == null || total == null) ? '-- / --' : '${current.toString().padLeft(2,'0')} / ${total.toString().padLeft(2,'0')}';
    final progress = (current != null && total != null && total! > 0) ? (current! / total!).clamp(0.0, 1.0) : 0.0;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      color: AppTheme.surface,
      child: Row(children: [
        const Text('LAP', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textDim)),
        const SizedBox(width: 10),
        Text(lapText, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w900, fontFeatures: [FontFeature.tabularFigures()], letterSpacing: 0.6, color: AppTheme.textPrimary)),
        const SizedBox(width: 12),
        Expanded(
          child: ClipRRect(
            borderRadius: BorderRadius.circular(99),
            child: SizedBox(
              height: 6,
              child: LinearProgressIndicator(
                value: progress,
                backgroundColor: AppTheme.surface3,
                valueColor: AlwaysStoppedAnimation(isRacing ? AppTheme.accent : AppTheme.textDim),
              ),
            ),
          ),
        ),
        const SizedBox(width: 10),
        Text('${(progress*100).toStringAsFixed(0)}%', style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.textDim)),
      ]),
    );
  }
}

class _RdBtn extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onPressed;
  const _RdBtn({required this.icon, required this.label, required this.onPressed});
  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(right: 2),
      child: Tooltip(
        message: label,
        child: InkWell(
          onTap: onPressed,
          borderRadius: BorderRadius.circular(8),
          child: Container(
            width: 36, height: 32,
            decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)),
            child: Icon(icon, size: 16, color: AppTheme.textSecondary),
          ),
        ),
      ),
    );
  }
}
