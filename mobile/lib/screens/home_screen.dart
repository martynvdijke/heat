import 'package:flutter/material.dart';
import '../theme/app_theme.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: CustomScrollView(
          slivers: [
            SliverToBoxAdapter(
              child: Container(
                padding: const EdgeInsets.fromLTRB(20, 24, 20, 20),
                decoration: const BoxDecoration(
                  color: AppTheme.bg,
                  border: Border(bottom: BorderSide(color: AppTheme.borderCol)),
                ),
                child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                  Row(children: [
                    Container(
                      width: 36, height: 36,
                      decoration: BoxDecoration(
                        color: AppTheme.accent,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: const Icon(Icons.flag, size: 20, color: Color(0xFF0B0B0B)),
                    ),
                    const SizedBox(width: 10),
                    const Text('HEAT', style: TextStyle(fontSize: 20, fontWeight: FontWeight.w900, letterSpacing: 2.5, color: AppTheme.textPrimary)),
                    const SizedBox(width: 10),
                    Container(width: 1, height: 18, color: AppTheme.borderStrong),
                    const SizedBox(width: 10),
                    const Text('RACE CONTROL', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 2.0, color: AppTheme.textDim)),
                    const Spacer(),
                    _HeaderPulse(),
                  ]),
                  const SizedBox(height: 16),
                  const Text('Companion', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w700, letterSpacing: 2.2, color: AppTheme.textDim)),
                  const SizedBox(height: 4),
                  Text('Telemetry & timing', style: Theme.of(context).textTheme.headlineMedium?.copyWith(fontSize: 22)),
                ]),
              ),
            ),
            SliverPadding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
              sliver: SliverList.list(children: [
                _NavCard(
                  accent: AppTheme.accent,
                  icon: Icons.sports_score,
                  title: 'LIVE RACE HUD',
                  subtitle: 'Leaderboard · laps · gaps · flag',
                  hint: 'HUD',
                  onTap: () => Navigator.pushNamed(context, '/hud'),
                ),
                const SizedBox(height: 10),
                _NavCard(
                  accent: const Color(0xFF2EC4B6),
                  icon: Icons.camera_alt_outlined,
                  title: 'ROUND SNAPSHOT',
                  subtitle: 'Capture & submit results',
                  hint: 'SHOT',
                  onTap: () => Navigator.pushNamed(context, '/snapshot'),
                ),
                const SizedBox(height: 10),
                _NavCard(
                  key: const Key('navStandings'),
                  accent: const Color(0xFF7C5CFF),
                  icon: Icons.leaderboard_outlined,
                  title: 'STANDINGS & STATS',
                  subtitle: 'Seasons · charts · .ics export',
                  hint: 'PTS',
                  onTap: () => Navigator.pushNamed(context, '/standings'),
                ),
                const SizedBox(height: 10),
                _NavCard(
                  accent: AppTheme.textDim,
                  icon: Icons.settings_outlined,
                  title: 'SETTINGS',
                  subtitle: 'Server URL · sign out',
                  hint: 'CFG',
                  onTap: () => Navigator.pushNamed(context, '/onboarding'),
                  muted: true,
                ),
                const SizedBox(height: 24),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  decoration: BoxDecoration(color: AppTheme.surface, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)),
                  child: Row(children: [
                    const Icon(Icons.info_outline, size: 14, color: AppTheme.textDim),
                    const SizedBox(width: 8),
                    Expanded(child: Text('All data via configured server. Check connection in Settings if offline.', style: Theme.of(context).textTheme.bodySmall)),
                  ]),
                ),
              ]),
            ),
          ],
        ),
      ),
    );
  }
}

class _HeaderPulse extends StatefulWidget {
  @override
  State<_HeaderPulse> createState() => _HeaderPulseState();
}
class _HeaderPulseState extends State<_HeaderPulse> with SingleTickerProviderStateMixin {
  late final AnimationController _c;
  @override
  void initState() { super.initState(); _c = AnimationController(vsync: this, duration: const Duration(milliseconds: 1400))..repeat(reverse: true); }
  @override
  void dispose() { _c.dispose(); super.dispose(); }
  @override
  Widget build(BuildContext context) {
    return FadeTransition(
      opacity: Tween(begin: 0.5, end: 1.0).animate(CurvedAnimation(parent: _c, curve: Curves.easeInOut)),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.borderCol)),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Container(width: 6, height: 6, decoration: const BoxDecoration(color: AppTheme.stateRacing, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          const Text('LIVE', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.2, color: AppTheme.textSecondary)),
        ]),
      ),
    );
  }
}

class _NavCard extends StatefulWidget {
  final Color accent;
  final IconData icon;
  final String title;
  final String subtitle;
  final String hint;
  final VoidCallback onTap;
  final bool muted;
  const _NavCard({super.key, required this.accent, required this.icon, required this.title, required this.subtitle, required this.hint, required this.onTap, this.muted = false});
  @override
  State<_NavCard> createState() => _NavCardState();
}
class _NavCardState extends State<_NavCard> {
  bool _pressed = false;
  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTapDown: (_) => setState(()=>_pressed=true),
      onTapCancel: () => setState(()=>_pressed=false),
      onTapUp: (_) => setState(()=>_pressed=false),
      onTap: widget.onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 120),
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: _pressed ? AppTheme.surface3 : AppTheme.surface2,
          borderRadius: BorderRadius.circular(AppTheme.radiusMd),
          border: Border.all(color: _pressed ? AppTheme.borderStrong : AppTheme.borderCol),
        ),
        child: Row(children: [
          Container(
            width: 44, height: 44,
            decoration: BoxDecoration(
              color: widget.muted ? AppTheme.surface3 : widget.accent.withValues(alpha: 0.14),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: widget.muted ? AppTheme.borderCol : widget.accent.withValues(alpha: 0.22)),
            ),
            child: Icon(widget.icon, size: 20, color: widget.muted ? AppTheme.textDim : widget.accent),
          ),
          const SizedBox(width: 14),
          Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(widget.title, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textPrimary)),
            const SizedBox(height: 2),
            Text(widget.subtitle, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
          ])),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(color: AppTheme.bgElevated, borderRadius: BorderRadius.circular(6), border: Border.all(color: AppTheme.borderCol)),
            child: Text(widget.hint, style: const TextStyle(fontSize: 9, fontWeight: FontWeight.w800, letterSpacing: 1.0, color: AppTheme.textDim)),
          ),
          const SizedBox(width: 8),
          const Icon(Icons.chevron_right, size: 18, color: AppTheme.textDim),
        ]),
      ),
    );
  }
}
