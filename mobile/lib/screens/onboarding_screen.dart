import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';
import '../theme/app_theme.dart';
import '../providers/session_providers.dart';

class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});
  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  final _urlCtrl = TextEditingController();
  final _userCtrl = TextEditingController();
  final _passCtrl = TextEditingController();
  final _tokenCtrl = TextEditingController();
  String _mode = 'director';
  bool _checking = false;
  String? _status;
  bool? _oidcEnabled;
  int? _selectedRacerId;
  String? _selectedRacerName;
  List<Map<String, dynamic>> _racers = [];

  @override
  void initState() {
    super.initState();
    final s = ref.read(sessionStoreProvider);
    if (s.baseUrl != null) _urlCtrl.text = s.baseUrl!;
  }

  String _normalizedBase(String raw) {
    var v = raw.trim();
    if (v.endsWith('/')) v = v.substring(0, v.length - 1);
    return v;
  }

  Future<void> _checkReachability() async {
    setState(() { _checking = true; _status = null; });
    final base = _normalizedBase(_urlCtrl.text);
    if (base.isEmpty) { setState(() { _status = 'Enter server URL'; _checking = false; }); return; }
    final store = ref.read(sessionStoreProvider);
    await store.setBaseUrl(base);
    final api = ref.read(apiClientProvider);
    try {
      await api.fetchRaceState();
      setState(() => _status = 'Server reachable');
    } catch (_) {
      try {
        await api.fetchRacers();
        setState(() => _status = 'Server reachable');
      } catch (e) {
        setState(() => _status = 'Cannot reach server: $e');
        _checking = false;
        return;
      }
    }
    final enabled = await api.checkOidcEnabled();
    try {
      final racers = await api.fetchRacers();
      _racers = racers.map((r) => {'id': r.id, 'name': r.name}).toList();
    } catch (_) {}
    setState(() { _oidcEnabled = enabled; _checking = false; });
  }

  Future<void> _loginDirector() async {
    setState(() { _checking = true; _status = null; });
    try {
      await ref.read(apiClientProvider).loginRaceDirector(_userCtrl.text.trim(), _passCtrl.text);
      if (mounted) Navigator.of(context).pushReplacementNamed('/home');
    } catch (e) {
      setState(() => _status = 'Login failed: $e');
    } finally {
      setState(() => _checking = false);
    }
  }

  Future<void> _loginPlayerPick() async {
    if (_selectedRacerId == null) { setState(() => _status = 'Pick a racer'); return; }
    setState(() { _checking = true; });
    try {
      await ref.read(apiClientProvider).loginPlayer(_selectedRacerId!, 'android');
      if (mounted) Navigator.of(context).pushReplacementNamed('/home');
    } catch (e) {
      setState(() => _status = 'Player login failed: $e');
    } finally {
      setState(() => _checking = false);
    }
  }

  Future<void> _validateToken() async {
    final t = _tokenCtrl.text.trim();
    if (t.isEmpty) return;
    setState(() { _checking = true; });
    try {
      final data = await ref.read(apiClientProvider).validatePlayerToken(t);
      final rid = (data['racer_id'] as num).toInt();
      final rname = data['racer_name'] as String;
      await ref.read(sessionStoreProvider).setPlayerToken(t, rid, rname);
      if (mounted) Navigator.of(context).pushReplacementNamed('/home');
    } catch (e) {
      setState(() => _status = 'Invalid token: $e');
    } finally {
      setState(() => _checking = false);
    }
  }

  Future<void> _openSso() async {
    final base = _normalizedBase(_urlCtrl.text);
    final url = Uri.parse('$base/api/auth/oidc/login');
    if (await canLaunchUrl(url)) await launchUrl(url, mode: LaunchMode.externalApplication);
  }

  @override
  Widget build(BuildContext context) {
    final reachable = _status != null && _status!.contains('reachable');
    final error = _status != null && !reachable;
    return Scaffold(
      backgroundColor: AppTheme.bg,
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 24),
          child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            // Header
            Row(children: [
              Container(width: 40, height: 40, decoration: BoxDecoration(color: AppTheme.accent, borderRadius: BorderRadius.circular(10)), child: const Icon(Icons.flag, color: Color(0xFF0B0B0B))),
              const SizedBox(width: 12),
              const Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('HEAT', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w900, letterSpacing: 2.4, color: AppTheme.textPrimary)),
                Text('SETUP  ·  RACE CONTROL', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1.4, color: AppTheme.textDim)),
              ]),
            ]),
            const SizedBox(height: 20),
            // Server card
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusLg), border: Border.all(color: AppTheme.borderCol)),
              child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                Row(children: [
                  Container(width: 28, height: 28, decoration: BoxDecoration(color: AppTheme.surface3, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)), child: const Icon(Icons.dns_outlined, size: 16, color: AppTheme.textSecondary)),
                  const SizedBox(width: 10),
                  const Text('SERVER', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textSecondary)),
                  const Spacer(),
                  if (reachable) Container(padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4), decoration: BoxDecoration(color: AppTheme.stateRacing.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.stateRacing.withValues(alpha: 0.24))), child: Row(mainAxisSize: MainAxisSize.min, children: [Container(width: 6, height: 6, decoration: const BoxDecoration(color: AppTheme.stateRacing, shape: BoxShape.circle)), const SizedBox(width: 6), const Text('ONLINE', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 0.8, color: AppTheme.stateRacing))])),
                ]),
                const SizedBox(height: 12),
                TextField(
                  key: const Key('serverUrlField'),
                  controller: _urlCtrl,
                  style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary),
                  decoration: const InputDecoration(labelText: 'Server URL (e.g. http://192.168.1.10:6270)', hintText: 'http://192.168.1.10:6270', border: OutlineInputBorder()),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  height: 44,
                  child: ElevatedButton(
                    key: const Key('connectButton'),
                    onPressed: _checking ? null : _checkReachability,
                    child: _checking
                        ? const SizedBox(height: 16, width: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Color(0xFF0B0B0B)))
                        : const Row(mainAxisAlignment: MainAxisAlignment.center, children: [Icon(Icons.wifi_tethering, size: 16), SizedBox(width: 8), Text('CHECK CONNECTION')]),
                  ),
                ),
                if (_status != null) ...[
                  const SizedBox(height: 10),
                  AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    decoration: BoxDecoration(
                      color: reachable ? AppTheme.stateRacing.withValues(alpha: 0.10) : error ? AppTheme.stateDnf.withValues(alpha: 0.10) : AppTheme.surface,
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: reachable ? AppTheme.stateRacing.withValues(alpha: 0.22) : error ? AppTheme.stateDnf.withValues(alpha: 0.22) : AppTheme.borderCol),
                    ),
                    child: Row(children: [
                      Icon(reachable ? Icons.check_circle : Icons.error_outline, size: 14, color: reachable ? AppTheme.stateRacing : AppTheme.stateDnf),
                      const SizedBox(width: 8),
                      Expanded(child: Text(_status!, style: TextStyle(fontSize: 12, color: reachable ? AppTheme.stateRacing : AppTheme.stateDnf))),
                    ]),
                  ),
                ],
              ]),
            ),
            const SizedBox(height: 16),
            // Auth card
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusLg), border: Border.all(color: AppTheme.borderCol)),
              child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                const Text('SIGN IN', style: TextStyle(fontSize: 11, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textSecondary)),
                const SizedBox(height: 12),
                SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'director', label: Text('Race Director'), icon: Icon(Icons.shield_outlined, size: 16)),
                    ButtonSegment(value: 'player', label: Text('Player Token'), icon: Icon(Icons.sports_motorsports_outlined, size: 16)),
                  ],
                  selected: {_mode},
                  onSelectionChanged: (s) => setState(() => _mode = s.first),
                ),
                // Stable key for E2E to switch to Player mode (tapping SegmentedButton segment is not key-addressable).
                InkWell(
                  key: const Key('playerLoginMode'),
                  onTap: () => setState(() => _mode = 'player'),
                  child: const SizedBox(width: 1, height: 1),
                ),
                const SizedBox(height: 16),
                AnimatedSwitcher(
                  duration: const Duration(milliseconds: 220),
                  child: _mode == 'director' ? _directorPane() : _playerPane(),
                ),
              ]),
            ),
          ]),
        ),
      ),
    );
  }

  Widget _directorPane() {
    return Column(key: const ValueKey('director'), crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      TextField(controller: _userCtrl, style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary), decoration: const InputDecoration(labelText: 'Username', hintText: 'director', border: OutlineInputBorder())),
      const SizedBox(height: 12),
      TextField(controller: _passCtrl, style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary), decoration: const InputDecoration(labelText: 'Password', border: OutlineInputBorder()), obscureText: true),
      const SizedBox(height: 12),
      SizedBox(height: 44, child: ElevatedButton(onPressed: _checking ? null : _loginDirector, child: const Text('SIGN IN AS RACE DIRECTOR'))),
      if (_oidcEnabled == true) ...[
        const SizedBox(height: 10),
        SizedBox(height: 44, child: OutlinedButton(onPressed: _openSso, child: const Row(mainAxisAlignment: MainAxisAlignment.center, children: [Icon(Icons.open_in_new, size: 14), SizedBox(width: 8), Text('SIGN IN WITH SSO')]))),
        const SizedBox(height: 8),
        const Text('SSO opens in external browser. Session cookie cannot be captured natively (out of scope).', style: TextStyle(fontSize: 11, color: AppTheme.textDim)),
      ],
    ]);
  }

  Widget _playerPane() {
    return Column(key: const ValueKey('player'), crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      if (_racers.isNotEmpty)
        DropdownButtonFormField<int>(
          key: const Key('racerField'),
          decoration: const InputDecoration(labelText: 'Pick racer', border: OutlineInputBorder()),
          dropdownColor: AppTheme.surface2,
          style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary),
          items: _racers.map((r) => DropdownMenuItem<int>(value: r['id'] as int, child: Text(r['name'] as String))).toList(),
          onChanged: (v) => setState(() { _selectedRacerId = v; _selectedRacerName = _racers.firstWhere((e) => e['id'] == v)['name'] as String; }),
        ),
      if (_racers.isNotEmpty) const SizedBox(height: 12),
      SizedBox(height: 44, child: ElevatedButton(key: const Key('loginButton'), onPressed: _checking ? null : _loginPlayerPick, child: Text(_selectedRacerName == null ? 'LOGIN AS SELECTED RACER' : 'LOGIN AS $_selectedRacerName'))),
      const SizedBox(height: 16),
      Container(height: 1, color: AppTheme.borderCol),
      const SizedBox(height: 12),
      const Text('Or paste existing token:', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1.0, color: AppTheme.textDim)),
      const SizedBox(height: 8),
      TextField(controller: _tokenCtrl, style: const TextStyle(fontSize: 13, color: AppTheme.textPrimary, fontFeatures: [FontFeature.tabularFigures()]), decoration: const InputDecoration(labelText: 'Player token', hintText: 'paste token', border: OutlineInputBorder())),
      const SizedBox(height: 10),
      SizedBox(height: 44, child: OutlinedButton(onPressed: _checking ? null : _validateToken, child: const Text('VALIDATE TOKEN & CONTINUE'))),
    ]);
  }
}
