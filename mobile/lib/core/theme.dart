// App theme: AIVIBE dark music-player theme.
// Mirrors web design tokens: obsidian surfaces, electric tangerine brand,
// gold reserved for verified/premium moments.
library;

import 'package:flutter/material.dart';

abstract final class AppTheme {
  static const brand = Color(0xFFFF5C00);
  static const brandStrong = Color(0xFFEA580C);
  static const gold = Color(0xFFFFC53D);
  static const surface0 = Color(0xFF09090B);
  static const surface1 = Color(0xFF111116);
  static const surface2 = Color(0xFF181820);
  static const surface3 = Color(0xFF22222D);
  static const surfaceBorder = Color(0xFF262633);

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

  /// Gold chip style for verified/premium moments.
  static BoxDecoration verifiedTagDecoration() => BoxDecoration(
        color: gold.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: gold.withValues(alpha: 0.4)),
      );
}
