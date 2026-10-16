import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import '../theme/app_theme.dart';
import '../providers/session_providers.dart';
import '../models/season.dart';
import '../models/round_snapshot.dart';
import '../services/api_client.dart';

class SnapshotScreen extends ConsumerStatefulWidget {
  const SnapshotScreen({super.key});
  @override
  ConsumerState<SnapshotScreen> createState() => _SnapshotScreenState();
}

class _SnapshotScreenState extends ConsumerState<SnapshotScreen> {
  final _raceNameCtrl = TextEditingController();
  List<Season> _seasons = [];
  int? _seasonId;
  XFile? _photo;
  Uint8List? _preview;
  bool _busy = false;
  String? _msg;
  bool _msgIsError = false;
  RoundSnapshot? _result;

  @override
  void initState() {
    super.initState();
    _loadSeasons();
  }

  Future<void> _loadSeasons() async {
    try {
      final s = await ref.read(apiClientProvider).fetchSeasons();
      setState(() => _seasons = s);
    } catch (_) {}
  }

  Future<void> _pick() async {
    final picker = ImagePicker();
    final f = await picker.pickImage(source: ImageSource.camera, maxWidth: 2048, maxHeight: 2048, imageQuality: 85);
    if (f == null) return;
    final bytes = await f.readAsBytes();
    setState(() { _photo = f; _preview = bytes; });
  }

  Future<void> _submit() async {
    if (_seasonId == null) { setState(() { _msg = 'Pick an active season (season_id required)'; _msgIsError = true; }); return; }
    if (_raceNameCtrl.text.trim().isEmpty) { setState(() { _msg = 'Enter race name'; _msgIsError = true; }); return; }
    setState(() { _busy = true; _msg = null; _result = null; _msgIsError = false; });
    try {
      final api = ref.read(apiClientProvider);
      String? uploadedUrl;
      if (_photo != null && ref.read(sessionStoreProvider).isRaceDirector) {
        final bytes = _preview ?? await _photo!.readAsBytes();
        uploadedUrl = await api.uploadImage(XFileLike(bytes, _photo!.name));
      }
      final created = await api.createRoundSnapshot(raceName: _raceNameCtrl.text.trim(), seasonId: _seasonId!);
      final id = (created['id'] as num).toInt();
      final snap = await api.fetchRoundSnapshot(id);
      setState(() { _result = snap; _msg = uploadedUrl != null ? 'Uploaded: $uploadedUrl' : 'Snapshot #${snap.round} created'; _msgIsError = false; });
    } catch (e) {
      setState(() { _msg = 'Failed: $e'; _msgIsError = true; });
    } finally {
      setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final seasons = _seasons.where((s) => s.status == 'active').toList();
    return Scaffold(
      backgroundColor: AppTheme.bg,
      appBar: AppBar(
        title: const Text('ROUND SNAPSHOT'),
        bottom: PreferredSize(preferredSize: const Size.fromHeight(1), child: Container(height: 1, color: AppTheme.borderCol)),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
        child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          // Header explainer
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(10), border: Border.all(color: AppTheme.borderCol)),
            child: Row(children: [
              Container(width: 30, height: 30, decoration: BoxDecoration(color: AppTheme.surface3, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)), child: const Icon(Icons.camera_alt_outlined, size: 16, color: AppTheme.textSecondary)),
              const SizedBox(width: 10),
              const Expanded(child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Text('CAPTURE  →  PREVIEW  →  SUBMIT', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.2, color: AppTheme.textSecondary)),
                SizedBox(height: 2),
                Text('Photo upload is best-effort (Race Director only). Snapshot always created.', style: TextStyle(fontSize: 11, color: AppTheme.textDim)),
              ])),
            ]),
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(AppTheme.radiusLg), border: Border.all(color: AppTheme.borderCol)),
            child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
              const Text('DETAILS', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 1.4, color: AppTheme.textDim)),
              const SizedBox(height: 10),
              DropdownButtonFormField<int>(
                decoration: const InputDecoration(labelText: 'SEASON (REQUIRED)', border: OutlineInputBorder()),
                dropdownColor: AppTheme.surface2,
                style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary),
                items: seasons.map((s) => DropdownMenuItem(value: s.id, child: Text(s.name))).toList(),
                onChanged: (v) => setState(() => _seasonId = v),
                initialValue: _seasonId,
              ),
              const SizedBox(height: 12),
              TextField(controller: _raceNameCtrl, style: const TextStyle(fontSize: 14, color: AppTheme.textPrimary), decoration: const InputDecoration(labelText: 'RACE NAME', hintText: 'e.g. Monaco GP — Round 4', border: OutlineInputBorder())),
              const SizedBox(height: 12),
              // Photo row
              Row(children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: _pick,
                    icon: const Icon(Icons.camera_alt, size: 16),
                    label: Text(_preview == null ? 'CAPTURE PHOTO' : 'RETAKE PHOTO'),
                  ),
                ),
                if (_preview != null) ...[
                  const SizedBox(width: 10),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                    decoration: BoxDecoration(color: AppTheme.accent.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.accent.withValues(alpha: 0.22))),
                    child: Row(mainAxisSize: MainAxisSize.min, children: [
                      Container(width: 6, height: 6, decoration: const BoxDecoration(color: AppTheme.accent, shape: BoxShape.circle)),
                      const SizedBox(width: 6),
                      Text('${(_preview!.length/1024).toStringAsFixed(0)} KB', style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w700, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.accent)),
                    ]),
                  ),
                ],
              ]),
              if (_preview != null) ...[
                const SizedBox(height: 12),
                ClipRRect(
                  borderRadius: BorderRadius.circular(AppTheme.radiusMd),
                  child: Container(
                    decoration: BoxDecoration(border: Border.all(color: AppTheme.borderCol), borderRadius: BorderRadius.circular(AppTheme.radiusMd)),
                    child: Image.memory(_preview!, height: 200, fit: BoxFit.contain),
                  ),
                ),
                const SizedBox(height: 8),
                const Row(children: [
                  Icon(Icons.info_outline, size: 12, color: AppTheme.textDim),
                  SizedBox(width: 6),
                  Expanded(child: Text('Best-effort upload — failure does not block snapshot creation.', style: TextStyle(fontSize: 11, color: AppTheme.textDim))),
                ]),
              ],
              const SizedBox(height: 14),
              SizedBox(
                height: 46,
                child: ElevatedButton(
                  onPressed: _busy ? null : _submit,
                  child: _busy
                      ? const SizedBox(height: 16, width: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Color(0xFF0B0B0B)))
                      : const Text('CREATE SNAPSHOT'),
                ),
              ),
              if (_msg != null) ...[
                const SizedBox(height: 10),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                  decoration: BoxDecoration(
                    color: _msgIsError ? AppTheme.stateDnf.withValues(alpha: 0.10) : AppTheme.stateRacing.withValues(alpha: 0.10),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: _msgIsError ? AppTheme.stateDnf.withValues(alpha: 0.22) : AppTheme.stateRacing.withValues(alpha: 0.22)),
                  ),
                  child: Row(children: [
                    Icon(_msgIsError ? Icons.error_outline : Icons.check_circle, size: 14, color: _msgIsError ? AppTheme.stateDnf : AppTheme.stateRacing),
                    const SizedBox(width: 8),
                    Expanded(child: Text(_msg!, style: TextStyle(fontSize: 12, color: _msgIsError ? AppTheme.stateDnf : AppTheme.stateRacing))),
                  ]),
                ),
              ],
            ]),
          ),
          if (_result != null) ...[
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(color: AppTheme.bgElevated, borderRadius: BorderRadius.circular(AppTheme.radiusLg), border: Border.all(color: AppTheme.borderStrong)),
              child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                Row(children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(color: AppTheme.accent, borderRadius: BorderRadius.circular(6)),
                    child: Text('ROUND ${_result!.round}', style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w900, letterSpacing: 0.8, color: Color(0xFF0B0B0B))),
                  ),
                  const SizedBox(width: 10),
                  Expanded(child: Text(_result!.raceName, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w700, color: AppTheme.textPrimary))),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                    decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(99), border: Border.all(color: AppTheme.borderCol)),
                    child: Text(_result!.status.toUpperCase(), style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.8, color: AppTheme.textDim)),
                  ),
                ]),
                const SizedBox(height: 12),
                const Divider(height: 1),
                const SizedBox(height: 8),
                ..._result!.scores.map((s) => Container(
                      margin: const EdgeInsets.only(bottom: 6),
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                      decoration: BoxDecoration(color: AppTheme.surface2, borderRadius: BorderRadius.circular(8), border: Border.all(color: AppTheme.borderCol)),
                      child: Row(children: [
                        Container(
                          width: 26, height: 22, alignment: Alignment.center,
                          decoration: BoxDecoration(color: AppTheme.surface3, borderRadius: BorderRadius.circular(6), border: Border.all(color: AppTheme.borderCol)),
                          child: Text('${s.position}', style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w800, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.textPrimary)),
                        ),
                        const SizedBox(width: 10),
                        Expanded(child: Text(s.racerName, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: AppTheme.textPrimary))),
                        if (s.dnf) _pill('DNF', AppTheme.stateDnf),
                        if (s.dns) _pill('DNS', AppTheme.textDim),
                        if (s.spins > 0) ...[const SizedBox(width: 6), Text('spins:${s.spins}', style: const TextStyle(fontSize: 11, color: AppTheme.textDim, fontFeatures: [FontFeature.tabularFigures()]))],
                        const SizedBox(width: 10),
                        Text('${s.points}', style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, fontFeatures: [FontFeature.tabularFigures()], color: AppTheme.accent)),
                        const SizedBox(width: 2),
                        const Text('PTS', style: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.6, color: AppTheme.textDim)),
                      ]),
                    )),
              ]),
            ),
          ],
        ]),
      ),
    );
  }

  Widget _pill(String t, Color c) => Container(
        margin: const EdgeInsets.only(left: 6),
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
        decoration: BoxDecoration(color: c.withValues(alpha: 0.14), borderRadius: BorderRadius.circular(99), border: Border.all(color: c.withValues(alpha: 0.28))),
        child: Text(t, style: TextStyle(fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 0.6, color: c)),
      );
}
