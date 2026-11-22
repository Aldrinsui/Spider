import 'package:flutter/material.dart';
import 'core/theme.dart';
import 'screens/map_search_screen.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const SpiderFreightApp());
}

class SpiderFreightApp extends StatelessWidget {
  const SpiderFreightApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Spider Freight',
      debugShowCheckedModeBanner: false,
      theme: SpiderTheme.theme,
      home: const MapSearchScreen(),
    );
  }
}
