// App theme: AIVIBE dark music-player theme.
// Mirrors web design tokens: near-black studio surfaces, signal-cyan brand.
library;

import 'package:flutter/material.dart';

abstract final class AppTheme {
  static const brand = Color(0xFF00D4FF);
  static const brandStrong = Color(0xFF00B8DB);
  static const brandInk = Color(0xFF04121A);
  static const surface0 = Color(0xFF08080D);
  static const surface1 = Color(0xFF0E0E15);
  static const surface2 = Color(0xFF16161E);
  static const surface3 = Color(0xFF202029);
  static const surfaceBorder = Color(0xFF23232E);

  static ThemeData get dark {
    final scheme = ColorScheme.fromSeed(
      seedColor: brand,
      brightness: Brightness.dark,
      surface: surface1,
    );
    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: surface0,
      appBarTheme: const AppBarTheme(centerTitle: false, elevation: 0),
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

  /// Cyan chip style for verified/premium moments.
  static BoxDecoration verifiedTagDecoration() => BoxDecoration(
        color: brand.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: brand.withValues(alpha: 0.4)),
      );
}
