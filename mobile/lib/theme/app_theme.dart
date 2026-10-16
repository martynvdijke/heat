import 'package:flutter/material.dart';

/// Heat — racing telemetry design tokens.
/// Dark scoreboard aesthetic: near-black graphite/blue depth, high-contrast
/// amber accent, semantic race states. Tabular numerals for timing data.
class AppTheme {
  // ── Core palette ──
  static const bg = Color(0xFF080A0F);
  static const bgElevated = Color(0xFF0F121C);
  static const surface = Color(0xFF131722);
  static const surface2 = Color(0xFF1A1F2E);
  static const surface3 = Color(0xFF222A3D);
  static const surface4 = Color(0xFF2A344E);
  static const borderCol = Color(0xFF1E2A3A);
  static const borderStrong = Color(0xFF2D3A4E);

  static const accent = Color(0xFFFFC300); // electric amber
  static const accentDim = Color(0xFFCC9A00);
  static const accentMuted = Color(0x331A1F2E); // not used directly
  static const accent2 = Color(0xFFFF3B30); // racing red secondary

  static const textPrimary = Color(0xFFF1F2F6);
  static const textSecondary = Color(0xFF9AA3B8);
  static const textDim = Color(0xFF5E6A84);

  // ── Semantic race states ──
  static const stateRacing = Color(0xFF00E676); // vivid green
  static const stateRacingBg = Color(0xFF0A1F14);
  static const statePaused = Color(0xFFFFB800);
  static const statePausedBg = Color(0xFF1F1A0A);
  static const stateStopped = Color(0xFF6B7280);
  static const stateStoppedBg = Color(0xFF15171A);
  static const stateDnf = Color(0xFFFF3B30);
  static const stateDns = Color(0xFF6B7280);

  // ── Radii & spacing ──
  static const radiusXs = 6.0;
  static const radiusSm = 8.0;
  static const radiusMd = 12.0;
  static const radiusLg = 16.0;
  static const radiusXl = 20.0;

  // ── Typography — tabular numerals for telemetry ──
  static const monoTabular = TextStyle(fontFeatures: [FontFeature.tabularFigures()]);

  static TextStyle get displayNum => const TextStyle(
        fontSize: 28,
        fontWeight: FontWeight.w800,
        letterSpacing: -0.5,
        fontFeatures: [FontFeature.tabularFigures()],
        color: textPrimary,
      );
  static TextStyle get lapNum => const TextStyle(
        fontSize: 18,
        fontWeight: FontWeight.w800,
        fontFeatures: [FontFeature.tabularFigures()],
        letterSpacing: 0.5,
        color: textPrimary,
      );

  static Color stateColor(String state) {
    switch (state.toLowerCase()) {
      case 'racing':
        return stateRacing;
      case 'paused':
        return statePaused;
      case 'stopped':
        return stateStopped;
      default:
        return textDim;
    }
  }

  static Color stateBg(String state) {
    switch (state.toLowerCase()) {
      case 'racing':
        return stateRacingBg;
      case 'paused':
        return statePausedBg;
      case 'stopped':
        return stateStoppedBg;
      default:
        return surface2;
    }
  }

  static ThemeData get dark {
    final scheme = ColorScheme.fromSeed(
      seedColor: accent,
      brightness: Brightness.dark,
      primary: accent,
      secondary: accent2,
      surface: surface,
      error: stateDnf,
    );
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.dark,
      scaffoldBackgroundColor: bg,
      colorScheme: scheme.copyWith(
        surface: surface,
        surfaceContainerHighest: surface2,
        primary: accent,
        onPrimary: const Color(0xFF0B0B0B),
        secondary: const Color(0xFF2EC4B6),
        error: stateDnf,
      ),
      appBarTheme: const AppBarTheme(
        backgroundColor: bg,
        foregroundColor: textPrimary,
        surfaceTintColor: Colors.transparent,
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 0,
        titleTextStyle: TextStyle(fontSize: 15, fontWeight: FontWeight.w700, letterSpacing: 1.2, color: textPrimary),
      ),
      cardTheme: CardThemeData(
        color: surface2,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusMd),
          side: const BorderSide(color: borderCol, width: 1),
        ),
      ),
      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: surface,
        contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
        hintStyle: const TextStyle(color: textDim, fontSize: 14),
        labelStyle: const TextStyle(color: textSecondary, fontSize: 12, fontWeight: FontWeight.w600, letterSpacing: 0.6),
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(radiusSm), borderSide: const BorderSide(color: borderCol)),
        enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(radiusSm), borderSide: const BorderSide(color: borderCol)),
        focusedBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(radiusSm), borderSide: const BorderSide(color: borderStrong, width: 1.2)),
        errorBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(radiusSm), borderSide: const BorderSide(color: stateDnf)),
      ),
      dividerTheme: const DividerThemeData(color: borderCol, thickness: 1, space: 1),
      chipTheme: ChipThemeData(
        backgroundColor: surface3,
        side: const BorderSide(color: borderCol),
        labelStyle: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: textSecondary),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(99)),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      ),
      textTheme: const TextTheme(
        headlineLarge: TextStyle(fontSize: 28, fontWeight: FontWeight.w800, letterSpacing: -0.8, color: textPrimary),
        headlineMedium: TextStyle(fontSize: 20, fontWeight: FontWeight.w700, letterSpacing: -0.4, color: textPrimary),
        titleLarge: TextStyle(fontSize: 13, fontWeight: FontWeight.w700, letterSpacing: 1.0, color: textSecondary),
        titleMedium: TextStyle(fontSize: 15, fontWeight: FontWeight.w600, color: textPrimary),
        titleSmall: TextStyle(fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 0.8, color: textSecondary),
        bodyLarge: TextStyle(fontSize: 15, fontWeight: FontWeight.w400, color: textPrimary, height: 1.4),
        bodyMedium: TextStyle(fontSize: 13, color: textSecondary, height: 1.45),
        bodySmall: TextStyle(fontSize: 11, color: textDim, letterSpacing: 0.3),
        labelLarge: TextStyle(fontSize: 13, fontWeight: FontWeight.w700, letterSpacing: 0.6, color: textPrimary),
        labelSmall: TextStyle(fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 1.0, color: textDim),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: accent,
          foregroundColor: const Color(0xFF0B0B0B),
          minimumSize: const Size(120, 44),
          elevation: 0,
          textStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, letterSpacing: 0.6),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusSm)),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          foregroundColor: textPrimary,
          side: const BorderSide(color: borderStrong),
          textStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.w700, letterSpacing: 0.5),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusSm)),
          minimumSize: const Size(120, 44),
        ),
      ),
      segmentedButtonTheme: SegmentedButtonThemeData(
        style: ButtonStyle(
          backgroundColor: WidgetStateProperty.resolveWith((states) => states.contains(WidgetState.selected) ? accent : surface2),
          foregroundColor: WidgetStateProperty.resolveWith((states) => states.contains(WidgetState.selected) ? const Color(0xFF0B0B0B) : textSecondary),
          side: const WidgetStatePropertyAll(BorderSide(color: borderCol)),
          textStyle: const WidgetStatePropertyAll(TextStyle(fontSize: 12, fontWeight: FontWeight.w700, letterSpacing: 0.6)),
          shape: WidgetStatePropertyAll(RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusSm))),
        ),
      ),
    );
  }
}
