import 'package:flutter/material.dart';

class SpiderTheme {
  static const Color darkBackground = Color(0xFF14171B);
  static const Color surfaceLight = Color(0xFFF2F0EA);
  static const Color brandOrange = Color(0xFFFF6A1A);
  static const Color mutedGray = Color(0xFF8FA3B3);
  static const Color darkBorder = Color(0xFF2A3038);
  static const Color successGreen = Color(0xFF10B981);
  static const Color dangerRed = Color(0xFFEF4444);

  static ThemeData get theme {
    return ThemeData(
      brightness: Brightness.dark,
      scaffoldBackgroundColor: darkBackground,
      primaryColor: brandOrange,
      colorScheme: const ColorScheme.dark(
        primary: brandOrange,
        surface: surfaceLight,
        background: darkBackground,
      ),
      fontFamily: 'Space Grotesk',
      appBarTheme: const AppBarTheme(
        backgroundColor: darkBackground,
        elevation: 0,
        centerTitle: false,
        titleTextStyle: TextStyle(
          color: surfaceLight,
          fontSize: 18,
          fontWeight: FontWeight.bold,
          letterSpacing: 0.5,
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: ElevatedButton.styleFrom(
          backgroundColor: brandOrange,
          foregroundColor: darkBackground,
          elevation: 0,
          shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
          padding: const EdgeInsets.symmetric(vertical: 16),
          textStyle: const TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w700,
            letterSpacing: 0.8,
          ),
        ),
      ),
    );
  }
}
