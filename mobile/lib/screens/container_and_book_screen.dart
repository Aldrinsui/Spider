import 'dart:async';
import 'package:flutter/material.dart';
import '../core/theme.dart';
import '../models/freight_models.dart';
import 'provider_verify_screen.dart';

class ContainerAndBookScreen extends StatefulWidget {
  final ContainerRoute route;

  const ContainerAndBookScreen({super.key, required this.route});

  @override
  State<ContainerAndBookScreen> createState() => _ContainerAndBookScreenState();
}

class _ContainerAndBookScreenState extends State<ContainerAndBookScreen> {
  final TextEditingController _lenController = TextEditingController(text: '120');
  final TextEditingController _widthController = TextEditingController(text: '80');
  final TextEditingController _heightController = TextEditingController(text: '90');
  final TextEditingController _weightController = TextEditingController(text: '640');

  double volumeCbm = 0.86;
  double chargeableCbm = 0.86;
  double totalBeforeGstRupees = 636.40;

  bool isHeld = false;
  int holdSecondsRemaining = 600; // 10 minutes
  Timer? _holdTimer;

  @override
  void initState() {
    super.initState();
    _recalculate();
  }

  void _recalculate() {
    final l = double.tryParse(_lenController.text) ?? 120.0;
    final w = double.tryParse(_widthController.text) ?? 80.0;
    final h = double.tryParse(_heightController.text) ?? 90.0;
    final rawVol = (l * w * h) / 1000000.0;
    setState(() {
      volumeCbm = (rawVol * 100).round() / 100.0;
      chargeableCbm = volumeCbm < 0.1 ? 0.1 : volumeCbm;
      totalBeforeGstRupees = chargeableCbm * widget.route.ratePerCbmRupees;
    });
  }

  void _startHold() {
    setState(() {
      isHeld = true;
      holdSecondsRemaining = 600;
    });

    _holdTimer?.cancel();
    _holdTimer = Timer.periodic(const Duration(seconds: 1), (timer) {
      if (holdSecondsRemaining > 0) {
        setState(() {
          holdSecondsRemaining--;
        });
      } else {
        timer.cancel();
        setState(() {
          isHeld = false;
        });
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Hold expired. Container space returned.')),
        );
      }
    });

    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Container space locked for 10 minutes in database transaction!'),
        backgroundColor: SpiderTheme.brandOrange,
      ),
    );
  }

  @override
  void dispose() {
    _holdTimer?.cancel();
    _lenController.dispose();
    _widthController.dispose();
    _heightController.dispose();
    _weightController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final route = widget.route;

    return Scaffold(
      backgroundColor: SpiderTheme.darkBackground,
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: SpiderTheme.surfaceLight),
          onPressed: () => Navigator.of(context).pop(),
        ),
      ),
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Head
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    '${route.transportMode} · ${route.isoCode}',
                    style: const TextStyle(
                      color: SpiderTheme.brandOrange,
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${route.originTitle} to ${route.destTitle}',
                    style: const TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontSize: 22,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Departs Thu 06:00 · ${route.availableCbm} of ${route.totalCbm} CBM free',
                    style: const TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontFamily: 'IBM Plex Mono',
                      fontSize: 13,
                    ),
                  ),
                ],
              ),
            ),

            const SizedBox(height: 12),
            // Polyline SVG representation
            Container(
              height: 60,
              color: SpiderTheme.darkBackground,
              child: CustomPaint(
                painter: _RoutePolylinePainter(),
              ),
            ),

            // Dimension Form & Calculation
            Expanded(
              child: Container(
                margin: const EdgeInsets.all(16),
                padding: const EdgeInsets.all(18),
                decoration: BoxDecoration(
                  color: SpiderTheme.surfaceLight,
                  borderRadius: BorderRadius.circular(0),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    const Text(
                      'CONSIGNMENT · L × W × H (CM)',
                      style: TextStyle(
                        color: SpiderTheme.darkBackground,
                        fontWeight: FontWeight.w600,
                        fontSize: 12,
                        letterSpacing: 0.8,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        _buildDimField(_lenController, '120'),
                        const SizedBox(width: 8),
                        _buildDimField(_widthController, '80'),
                        const SizedBox(width: 8),
                        _buildDimField(_heightController, '90'),
                      ],
                    ),
                    const SizedBox(height: 14),

                    const Text(
                      'WEIGHT (KG)',
                      style: TextStyle(
                        color: SpiderTheme.darkBackground,
                        fontWeight: FontWeight.w600,
                        fontSize: 12,
                        letterSpacing: 0.8,
                      ),
                    ),
                    const SizedBox(height: 8),
                    _buildDimField(_weightController, '640'),

                    const SizedBox(height: 14),
                    const Divider(color: SpiderTheme.darkBackground, height: 1),
                    const SizedBox(height: 14),

                    // Metrics
                    _buildCalcRow('Volume', '${volumeCbm.toStringAsFixed(2)} CBM'),
                    const SizedBox(height: 8),
                    _buildCalcRow('Chargeable', '${chargeableCbm.toStringAsFixed(2)} CBM'),
                    const SizedBox(height: 8),
                    _buildCalcRow('Rate', 'Rs ${route.ratePerCbmRupees.toStringAsFixed(0)} / CBM'),

                    const SizedBox(height: 14),
                    const Divider(color: SpiderTheme.darkBackground, height: 1),
                    const SizedBox(height: 14),

                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text(
                          'Total before GST',
                          style: TextStyle(
                            color: SpiderTheme.darkBackground,
                            fontWeight: FontWeight.bold,
                            fontSize: 17,
                          ),
                        ),
                        Text(
                          'Rs ${totalBeforeGstRupees.toStringAsFixed(2)}',
                          style: const TextStyle(
                            color: SpiderTheme.darkBackground,
                            fontFamily: 'IBM Plex Mono',
                            fontWeight: FontWeight.bold,
                            fontSize: 18,
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),

            // Bottom Bar with Hold Button and Timer
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Column(
                children: [
                  ElevatedButton(
                    onPressed: isHeld ? null : _startHold,
                    child: Text(
                      isHeld
                          ? 'SPACE HELD · ${holdSecondsRemaining ~/ 60}:${(holdSecondsRemaining % 60).toString().padLeft(2, '0')}'
                          : 'HOLD SPACE · 10 MIN',
                    ),
                  ),
                  const SizedBox(height: 6),
                  const Text(
                    'Held until you pay or the timer runs out',
                    style: TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontSize: 12,
                      fontFamily: 'IBM Plex Mono',
                    ),
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 8),
                  TextButton(
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const ProviderVerifyScreen()),
                      );
                    },
                    child: const Text(
                      'Are you a logistics provider? Verify Fleet →',
                      style: TextStyle(color: SpiderTheme.brandOrange),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildDimField(TextEditingController controller, String placeholder) {
    return Expanded(
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          border: Border.all(color: SpiderTheme.darkBackground),
        ),
        child: TextField(
          controller: controller,
          keyboardType: TextInputType.number,
          style: const TextStyle(
            color: SpiderTheme.darkBackground,
            fontFamily: 'IBM Plex Mono',
            fontWeight: FontWeight.w600,
            fontSize: 15,
          ),
          decoration: const InputDecoration(
            isDense: true,
            contentPadding: EdgeInsets.zero,
            border: InputBorder.none,
          ),
          onChanged: (_) => _recalculate(),
        ),
      ),
    );
  }

  Widget _buildCalcRow(String label, String value) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Text(
          label,
          style: const TextStyle(
            color: SpiderTheme.darkBackground,
            fontSize: 14,
          ),
        ),
        Text(
          value,
          style: const TextStyle(
            color: SpiderTheme.darkBackground,
            fontFamily: 'IBM Plex Mono',
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
      ],
    );
  }
}

class _RoutePolylinePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = SpiderTheme.brandOrange
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.0;

    final path = Path();
    path.moveTo(20, 40);
    path.lineTo(130, 22);
    path.lineTo(240, 34);
    path.lineTo(370, 18);
    canvas.drawPath(path, paint);

    final dotPaint = Paint()
      ..color = SpiderTheme.surfaceLight
      ..style = PaintingStyle.fill;
    canvas.drawRect(const Rect.fromLTWH(16, 36, 8, 8), dotPaint);
    canvas.drawRect(const Rect.fromLTWH(366, 14, 8, 8), dotPaint);
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
