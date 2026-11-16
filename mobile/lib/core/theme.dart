// App theme: AVIBE dark music-player theme.
// Mirrors web/app/assets/css/main.css design tokens v2 exactly:
// near-black studio surfaces, signal-cyan brand, status + text ramps.
library;

import 'package:flutter/material.dart';

abstract final class AppTheme {
  // Brand (signal cyan) — web --color-brand*.
  static const brand = Color(0xFF00D4FF);
  static const brandStrong = Color(0xFF00B8DB);
  static const brandSoft = Color(0xFF083344);
  static const brandInk = Color(0xFF04121A);
  static const brandDeep = Color(0xFF0A141B);
  static const brandDeepRaised = Color(0xFF10232E);

  // Surfaces — web --color-surface-*.
  static const surface0 = Color(0xFF08080D);
  static const surface1 = Color(0xFF0E0E15);
  static const surface2 = Color(0xFF16161E);
  static const surface3 = Color(0xFF202029);
  static const surfaceBorder = Color(0xFF23232E);
  static const surfaceBorderHover = Color(0xFF3A3A46);

  // Text — web --color-text-*.
  static const textPrimary = Color(0xFFF7F7F8);
  static const textSecondary = Color(0xFFA1A7B3);
  static const textMuted = Color(0xFF8A909C);

  // Status — web --color-success/warning/error/info.
  static const success = Color(0xFF22C55E);
  static const warning = Color(0xFFF59E0B);
  static const error = Color(0xFFEF4444);
  static const info = Color(0xFF00D4FF);

  /// Verified seal ONLY (web --color-verified) — never for decoration
  /// or generic success.
  static const verified = Color(0xFF10B981);

  /// Ink on brand-cyan surfaces (web --color-on-brand).
  static const onBrand = Color(0xFF04121A);

  /// Keyboard focus ring (web --color-focus).
  static const focus = Color(0xFF7DE8FF);

  // Shape — web --radius-*.
  static const radiusXs = 4.0;
  static const radiusSm = 8.0;
  static const radiusMd = 12.0;
  static const radiusLg = 16.0;
  static const radiusXl = 24.0;

  /// CJK + Thai fallback chain (WI-R12). The app ships no font files;
  /// each entry resolves to the platform system font so Korean,
  /// Japanese, Simplified Chinese, and Thai always render.
  static const fontFallbacks = [
    'Pretendard Variable',
    'Apple SD Gothic Neo',
    'Noto Sans KR',
    'Noto Sans JP',
    'Noto Sans SC',
    'Noto Sans Thai',
    'sans-serif',
  ];

  static ThemeData get dark {
    final scheme = ColorScheme.fromSeed(
      seedColor: brand,
      brightness: Brightness.dark,
      surface: surface1,
    ).copyWith(
      primary: brand,
      onPrimary: onBrand,
      secondary: textSecondary,
      surface: surface1,
      error: error,
      onSurface: textPrimary,
      onSurfaceVariant: textSecondary,
    );
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: surface0,
      fontFamilyFallback: fontFallbacks,
      textTheme: const TextTheme(
        bodyLarge: TextStyle(color: textPrimary),
        bodyMedium: TextStyle(color: textPrimary),
        bodySmall: TextStyle(color: textSecondary),
        titleLarge: TextStyle(color: textPrimary),
        titleMedium: TextStyle(color: textPrimary),
        titleSmall: TextStyle(color: textPrimary),
      ),
      appBarTheme: const AppBarTheme(
        centerTitle: false,
        elevation: 0,
        backgroundColor: surface0,
        foregroundColor: textPrimary,
      ),
      cardTheme: CardThemeData(
        color: surface1,
        elevation: 0,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: const BorderSide(color: surfaceBorder),
        ),
      ),
      sliderTheme: SliderThemeData(
        overlayShape: SliderComponentShape.noOverlay,
        trackHeight: 3,
        thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6),
      ),
      chipTheme: const ChipThemeData(
        shape: StadiumBorder(),
        side: BorderSide.none,
      ),
    );
  }

  /// Accent chip style for the AI-generated tag.
  static BoxDecoration aiTagDecoration(ColorScheme scheme) => BoxDecoration(
        color: scheme.primaryContainer,
        borderRadius: BorderRadius.circular(999),
      );

  /// Verified-seal chip (web --color-verified). Verified moments only.
  static BoxDecoration verifiedTagDecoration() => BoxDecoration(
        color: verified.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: verified.withValues(alpha: 0.4)),
      );
}
