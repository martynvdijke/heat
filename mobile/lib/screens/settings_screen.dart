import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';
import '../providers/session_providers.dart';
import '../theme/app_theme.dart';

/// Settings, legal links and the two self-service account deletion paths.
///
/// Both deletion actions require the user to type DELETE, matching the server
/// contract. The racer path deletes the racer identity; the admin path deletes
/// the race-director account. Neither touches historical race data.
class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  bool _busy = false;

  String get _base {
    final saved = ref.read(sessionStoreProvider).baseUrl;
    if (saved == null || saved.isEmpty) return 'https://heat.vandijke.xyz';
    return saved.endsWith('/') ? saved.substring(0, saved.length - 1) : saved;
  }

  Future<void> _openLegal(String path) async {
    final uri = Uri.parse('$_base$path');
    final ok = await launchUrl(uri, mode: LaunchMode.externalApplication);
    if (!ok && mounted) {
      _snack('Could not open $uri');
    }
  }

  Future<String?> _typedConfirmation({required String title, required String body}) async {
    final controller = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDialogState) => AlertDialog(
          backgroundColor: AppTheme.surface2,
          title: Text(title),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(body, style: const TextStyle(fontSize: 13, color: AppTheme.textSecondary)),
              const SizedBox(height: 14),
              TextField(
                key: const Key('deleteConfirmField'),
                controller: controller,
                autofocus: true,
                style: const TextStyle(color: AppTheme.textPrimary),
                decoration: const InputDecoration(
                  labelText: 'Type DELETE to confirm',
                  labelStyle: TextStyle(color: AppTheme.textDim),
                ),
                onChanged: (_) => setDialogState(() {}),
              ),
            ],
          ),
          actions: [
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
            TextButton(
              key: const Key('confirmDeleteButton'),
              onPressed: controller.text.trim().toUpperCase() == 'DELETE'
                  ? () => Navigator.pop(ctx, 'DELETE')
                  : null,
              child: const Text('Delete', style: TextStyle(color: AppTheme.accent2)),
            ),
          ],
        ),
      ),
    );
    controller.dispose();
    return result;
  }

  Future<void> _deleteRacer() async {
    final confirmed = await _typedConfirmation(
      title: 'Delete racer identity',
      body: 'This permanently deletes your racer profile, email sign-in links, Telegram link, player sessions, upgrades, achievements and statistics. '
          'Historical race results are kept for the championship but detached from you and stored as "Deleted racer". This cannot be undone.',
    );
    if (confirmed == null || !mounted) return;
    setState(() => _busy = true);
    try {
      await ref.read(apiClientProvider).deleteMe(confirmed);
      await ref.read(sessionStoreProvider).clearAuth();
      if (!mounted) return;
      Navigator.pushNamedAndRemoveUntil(context, '/onboarding', (_) => false);
    } catch (e) {
      if (mounted) _snack('$e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _deleteAdmin() async {
    final confirmed = await _typedConfirmation(
      title: 'Delete admin account',
      body: 'This permanently deletes the race-director account and revokes every admin session. Race, racer and result data are not deleted. '
          'The last remaining admin cannot be deleted. This cannot be undone.',
    );
    if (confirmed == null || !mounted) return;
    setState(() => _busy = true);
    try {
      await ref.read(apiClientProvider).deleteAccount(confirmed);
      await ref.read(sessionStoreProvider).clearAuth();
      if (!mounted) return;
      Navigator.pushNamedAndRemoveUntil(context, '/onboarding', (_) => false);
    } catch (e) {
      if (mounted) _snack('$e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _showExport(String label, Future<String> Function() load) async {
    setState(() => _busy = true);
    try {
      final json = await load();
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (ctx) => AlertDialog(
          backgroundColor: AppTheme.surface2,
          title: Text('$label export'),
          content: SizedBox(
            width: double.maxFinite,
            height: 360,
            child: SingleChildScrollView(child: SelectableText(json, style: const TextStyle(fontSize: 11, color: AppTheme.textSecondary))),
          ),
          actions: [
            TextButton(
              onPressed: () {
                Clipboard.setData(ClipboardData(text: json));
                Navigator.pop(ctx);
                _snack('Export copied to clipboard');
              },
              child: const Text('Copy'),
            ),
            TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Close')),
          ],
        ),
      );
    } catch (e) {
      if (mounted) _snack('Export failed: $e');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _signOut() async {
    await ref.read(apiClientProvider).logout();
    if (!mounted) return;
    Navigator.pushNamedAndRemoveUntil(context, '/onboarding', (_) => false);
  }

  void _snack(String message) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    final store = ref.watch(sessionStoreProvider);
    return Scaffold(
      appBar: AppBar(
        backgroundColor: AppTheme.bg,
        foregroundColor: AppTheme.textPrimary,
        title: const Text('SETTINGS', style: TextStyle(fontSize: 14, fontWeight: FontWeight.w800, letterSpacing: 2)),
      ),
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
          children: [
            _Section(
              title: 'SERVER',
              children: [
                _ValueRow(label: 'URL', value: store.baseUrl ?? _base),
                _ValueRow(label: 'Signed in as', value: store.isRaceDirector ? 'Race director' : (store.isPlayer ? (store.playerRacerName ?? 'Racer') : 'Not signed in')),
              ],
            ),
            const SizedBox(height: 14),
            _Section(
              title: 'LEGAL',
              children: [
                _LinkRow(icon: Icons.shield_outlined, label: 'Privacy notice', onTap: () => _openLegal('/privacy')),
                _LinkRow(icon: Icons.description_outlined, label: 'Terms of use', onTap: () => _openLegal('/terms')),
                _LinkRow(icon: Icons.info_outline, label: 'Account deletion instructions', onTap: () => _openLegal('/account/delete')),
              ],
            ),
            const SizedBox(height: 14),
            _Section(
              title: 'YOUR DATA',
              children: [
                if (store.isPlayer)
                  _LinkRow(icon: Icons.download_outlined, label: 'Export racer data', onTap: _busy ? null : () => _showExport('Racer', () => ref.read(apiClientProvider).exportMe())),
                if (store.isRaceDirector)
                  _LinkRow(icon: Icons.download_outlined, label: 'Export admin data', onTap: _busy ? null : () => _showExport('Admin', () => ref.read(apiClientProvider).exportAccount())),
                if (!store.isPlayer && !store.isRaceDirector) const _ValueRow(label: 'Nothing to export', value: 'Not signed in'),
              ],
            ),
            const SizedBox(height: 14),
            _Section(
              title: 'DELETE ACCOUNT',
              children: [
                if (store.isPlayer)
                  _LinkRow(
                    icon: Icons.person_remove_outlined,
                    label: 'Delete racer identity',
                    destructive: true,
                    onTap: _busy ? null : _deleteRacer,
                  ),
                if (store.isRaceDirector)
                  _LinkRow(
                    icon: Icons.admin_panel_settings_outlined,
                    label: 'Delete admin account',
                    destructive: true,
                    onTap: _busy ? null : _deleteAdmin,
                  ),
                if (!store.isPlayer && !store.isRaceDirector) const _ValueRow(label: 'Sign in first', value: 'No account on this device'),
                const Padding(
                  padding: EdgeInsets.fromLTRB(14, 4, 14, 12),
                  child: Text(
                    'Deletion is immediate and irreversible. Historical race results are kept but anonymised. '
                    'Backups may retain the old data until their retention period ends.',
                    style: TextStyle(fontSize: 11, color: AppTheme.textDim, height: 1.4),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),
            OutlinedButton.icon(
              key: const Key('signOutButton'),
              onPressed: _busy ? null : _signOut,
              icon: const Icon(Icons.logout, size: 18),
              label: const Text('SIGN OUT'),
            ),
          ],
        ),
      ),
    );
  }
}

class _Section extends StatelessWidget {
  final String title;
  final List<Widget> children;
  const _Section({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: AppTheme.surface2,
        borderRadius: BorderRadius.circular(AppTheme.radiusMd),
        border: Border.all(color: AppTheme.borderCol),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(14, 12, 14, 4),
            child: Text(title, style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.8, color: AppTheme.textDim)),
          ),
          ...children,
        ],
      ),
    );
  }
}

class _ValueRow extends StatelessWidget {
  final String label;
  final String value;
  const _ValueRow({required this.label, required this.value});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Row(
        children: [
          Text(label, style: const TextStyle(fontSize: 12, color: AppTheme.textSecondary)),
          const Spacer(),
          Flexible(child: Text(value, textAlign: TextAlign.right, style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: AppTheme.textPrimary))),
        ],
      ),
    );
  }
}

class _LinkRow extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback? onTap;
  final bool destructive;
  const _LinkRow({required this.icon, required this.label, required this.onTap, this.destructive = false});

  @override
  Widget build(BuildContext context) {
    final color = destructive ? AppTheme.accent2 : AppTheme.textPrimary;
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
        child: Row(
          children: [
            Icon(icon, size: 18, color: destructive ? AppTheme.accent2 : AppTheme.textDim),
            const SizedBox(width: 12),
            Expanded(child: Text(label, style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: color))),
            if (onTap != null) const Icon(Icons.chevron_right, size: 18, color: AppTheme.textDim),
          ],
        ),
      ),
    );
  }
}
