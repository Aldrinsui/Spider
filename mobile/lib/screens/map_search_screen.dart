import 'package:flutter/material.dart';
import '../core/theme.dart';
import '../models/freight_models.dart';
import 'container_and_book_screen.dart';

class MapSearchScreen extends StatefulWidget {
  const MapSearchScreen({super.key});

  @override
  State<MapSearchScreen> createState() => _MapSearchScreenState();
}

class _MapSearchScreenState extends State<MapSearchScreen> {
  // Mock routes matching HTML screen 1
  final List<ContainerRoute> routes = [
    ContainerRoute(
      id: 'msku-482913',
      isoCode: 'MSKU 482913',
      transportMode: 'Rail',
      originTitle: 'Chennai Central',
      destTitle: 'Bengaluru Inland Depot',
      departureTime: DateTime.now().add(const Duration(days: 2)),
      totalCbm: 30.0,
      availableCbm: 18.6,
      totalWeightKg: 20000.0,
      availableWeightKg: 14000.0,
      ratePerCbmPaise: 74000, // Rs 740
    ),
    ContainerRoute(
      id: 'tclu-110452',
      isoCode: 'TCLU 110452',
      transportMode: 'Road',
      originTitle: 'Ambattur hub',
      destTitle: 'Whitefield Depot',
      departureTime: DateTime.now().add(const Duration(days: 3)),
      totalCbm: 14.0,
      availableCbm: 3.4,
      totalWeightKg: 10000.0,
      availableWeightKg: 2500.0,
      ratePerCbmPaise: 61000, // Rs 610
    ),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: SpiderTheme.darkBackground,
      body: Stack(
        children: [
          // 1. Vector Map Canvas
          Positioned.fill(
            child: CustomPaint(
              painter: _MapGridPainter(),
            ),
          ),

          // 2. Top Location Bar: "Chennai Port → BLR"
          Positioned(
            top: 56,
            left: 16,
            right: 16,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
              decoration: BoxDecoration(
                color: SpiderTheme.darkBackground,
                border: Border.all(color: SpiderTheme.surfaceLight, width: 1),
              ),
              child: const Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'Chennai Port',
                    style: TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontSize: 15,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                  Text(
                    '→ BLR',
                    style: TextStyle(
                      color: SpiderTheme.brandOrange,
                      fontSize: 15,
                      fontFamily: 'IBM Plex Mono',
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),
          ),

          // 3. Bottom Freight Panel
          Positioned(
            bottom: 0,
            left: 0,
            right: 0,
            child: Container(
              padding: const EdgeInsets.all(18),
              decoration: const BoxDecoration(
                color: SpiderTheme.surfaceLight,
                border: Border(
                  top: BorderSide(color: SpiderTheme.brandOrange, width: 3),
                ),
              ),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  // Title row
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        '3 routes this week',
                        style: TextStyle(
                          color: SpiderTheme.darkBackground,
                          fontSize: 20,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                        decoration: BoxDecoration(
                          border: Border.all(color: SpiderTheme.darkBackground),
                        ),
                        child: const Text(
                          'SORT: DEPARTURE',
                          style: TextStyle(
                            color: SpiderTheme.darkBackground,
                            fontSize: 11,
                            letterSpacing: 0.8,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 12),
                  const Divider(color: SpiderTheme.darkBackground, height: 1),
                  const SizedBox(height: 12),

                  // Route 1 (Rail · Chennai Central)
                  _buildRouteCard(
                    title: 'Rail · Chennai Central',
                    rate: 'Rs 740/CBM',
                    subtitle: 'MSKU 482913 · Thu 06:00',
                    capacity: '18.6 / 30 CBM',
                    onTap: () => _openRoute(routes[0]),
                  ),

                  const SizedBox(height: 12),
                  const Divider(color: SpiderTheme.darkBackground, height: 1),
                  const SizedBox(height: 12),

                  // Route 2 (Road · Ambattur hub)
                  _buildRouteCard(
                    title: 'Road · Ambattur hub',
                    rate: 'Rs 610/CBM',
                    subtitle: 'TCLU 110452 · Fri 14:30',
                    capacity: '3.4 / 14 CBM',
                    onTap: () => _openRoute(routes[1]),
                  ),

                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () => _openRoute(routes[0]),
                    child: const Text('OPEN BEST ROUTE'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildRouteCard({
    required String title,
    required String rate,
    required String subtitle,
    required String capacity,
    required VoidCallback onTap,
  }) {
    return InkWell(
      onTap: onTap,
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                title,
                style: const TextStyle(
                  color: SpiderTheme.darkBackground,
                  fontWeight: FontWeight.bold,
                  fontSize: 14,
                ),
              ),
              Text(
                rate,
                style: const TextStyle(
                  color: SpiderTheme.darkBackground,
                  fontFamily: 'IBM Plex Mono',
                  fontWeight: FontWeight.bold,
                  fontSize: 14,
                ),
              ),
            ],
          ),
          const SizedBox(height: 4),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(
                subtitle,
                style: const TextStyle(
                  color: SpiderTheme.darkBackground,
                  fontFamily: 'IBM Plex Mono',
                  fontSize: 13,
                ),
              ),
              Text(
                capacity,
                style: const TextStyle(
                  color: SpiderTheme.darkBackground,
                  fontFamily: 'IBM Plex Mono',
                  fontSize: 13,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  void _openRoute(ContainerRoute route) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => ContainerAndBookScreen(route: route),
      ),
    );
  }
}

class _MapGridPainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paintGrid = Paint()
      ..color = const Color(0xFF2A3038)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.0;

    // Draw wavy contour lines like HTML mockup SVG
    for (double y = 120; y < size.height - 200; y += 80) {
      final path = Path();
      path.moveTo(-20, y);
      path.cubicTo(80, y - 30, 150, y + 40, 260, y + 10);
      path.cubicTo(320, y - 10, 390, y + 20, size.width + 20, y - 20);
      canvas.drawPath(path, paintGrid);
    }

    // Active corridor route in dashed orange
    final routePaint = Paint()
      ..color = SpiderTheme.brandOrange
      ..style = PaintingStyle.stroke
      ..strokeWidth = 3.0;

    final routePath = Path();
    routePath.moveTo(120, 80);
    routePath.lineTo(120, 200);
    routePath.lineTo(230, 230);
    routePath.lineTo(230, 360);
    routePath.lineTo(330, 390);
    canvas.drawPath(routePath, routePaint);

    // Nodes
    final nodePaint = Paint()..style = PaintingStyle.fill;

    nodePaint.color = SpiderTheme.brandOrange;
    canvas.drawRect(const Rect.fromLTWH(111, 71, 18, 18), nodePaint);

    nodePaint.color = SpiderTheme.surfaceLight;
    canvas.drawRect(const Rect.fromLTWH(221, 221, 18, 18), nodePaint);
    canvas.drawRect(const Rect.fromLTWH(321, 381, 18, 18), nodePaint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
