import 'package:flutter/material.dart';
import '../core/theme.dart';
import 'admin_review_screen.dart';

class ProviderVerifyScreen extends StatefulWidget {
  const ProviderVerifyScreen({super.key});

  @override
  State<ProviderVerifyScreen> createState() => _ProviderVerifyScreenState();
}

class _ProviderVerifyScreenState extends State<ProviderVerifyScreen> {
  bool frontUploaded = true;
  bool sideUploaded = false;
  bool doorsUploaded = false;

  @override
  Widget build(BuildContext context) {
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
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    'Provider · Step 2 of 3',
                    style: TextStyle(
                      color: SpiderTheme.brandOrange,
                      fontWeight: FontWeight.bold,
                      fontSize: 14,
                    ),
                  ),
                  SizedBox(height: 4),
                  Text(
                    'Verify your fleet',
                    style: TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontSize: 22,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 18),

            // Checklist Item 01: Transport Permit
            _buildChecklistItem(
              idx: '01',
              title: 'Transport permit',
              subtitle: 'permit_0417.pdf',
              isDone: true,
            ),
            const Divider(color: SpiderTheme.darkBorder, height: 1),

            // Checklist Item 02: Fleet RC
            _buildChecklistItem(
              idx: '02',
              title: 'Fleet RC',
              subtitle: 'rc_TN09.pdf',
              isDone: true,
            ),
            const Divider(color: SpiderTheme.darkBorder, height: 1),

            // Checklist Item 03: Three container photos
            _buildChecklistItem(
              idx: '03',
              title: 'Three container photos',
              subtitle: 'Geotagged on capture',
              isDone: false,
            ),
            const SizedBox(height: 14),

            // Photo Slots (Front, Side, Doors)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 18),
              child: Row(
                children: [
                  _buildPhotoSlot('Front', frontUploaded, () {
                    setState(() => frontUploaded = !frontUploaded);
                  }),
                  const SizedBox(width: 8),
                  _buildPhotoSlot('Side', sideUploaded, () {
                    setState(() => sideUploaded = !sideUploaded);
                  }),
                  const SizedBox(width: 8),
                  _buildPhotoSlot('Doors', doorsUploaded, () {
                    setState(() => doorsUploaded = !doorsUploaded);
                  }),
                ],
              ),
            ),

            const Spacer(),

            // Bottom CTA
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 20),
              child: Column(
                children: [
                  ElevatedButton(
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(
                          content: Text('KYC documents submitted for admin approval queue!'),
                          backgroundColor: SpiderTheme.brandOrange,
                        ),
                      );
                    },
                    child: const Text('SUBMIT FOR REVIEW'),
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Review takes up to 24 hours',
                    style: TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontSize: 12,
                      fontFamily: 'IBM Plex Mono',
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextButton(
                    onPressed: () {
                      Navigator.of(context).push(
                        MaterialPageRoute(builder: (_) => const AdminReviewScreen()),
                      );
                    },
                    child: const Text(
                      'Switch to Admin Review Queue Demo →',
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

  Widget _buildChecklistItem({
    required String idx,
    required String title,
    required String subtitle,
    required bool isDone,
  }) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
      child: Row(
        children: [
          Text(
            idx,
            style: const TextStyle(
              color: SpiderTheme.brandOrange,
              fontFamily: 'IBM Plex Mono',
              fontWeight: FontWeight.bold,
              fontSize: 16,
            ),
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    color: SpiderTheme.surfaceLight,
                    fontWeight: FontWeight.bold,
                    fontSize: 15,
                  ),
                ),
                const SizedBox(height: 2),
                Text(
                  subtitle,
                  style: const TextStyle(
                    color: SpiderTheme.mutedGray,
                    fontFamily: 'IBM Plex Mono',
                    fontSize: 12,
                  ),
                ),
              ],
            ),
          ),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            decoration: BoxDecoration(
              border: Border.all(
                color: isDone ? SpiderTheme.surfaceLight : SpiderTheme.brandOrange,
              ),
              color: isDone ? Colors.transparent : SpiderTheme.brandOrange.withOpacity(0.1),
            ),
            child: Text(
              isDone ? 'DONE' : 'OPEN',
              style: TextStyle(
                color: isDone ? SpiderTheme.surfaceLight : SpiderTheme.brandOrange,
                fontSize: 11,
                fontFamily: 'IBM Plex Mono',
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPhotoSlot(String label, bool uploaded, VoidCallback onTap) {
    return Expanded(
      child: InkWell(
        onTap: onTap,
        child: Container(
          height: 80,
          decoration: BoxDecoration(
            color: const Color(0xFF1E2228),
            border: Border.all(
              color: uploaded ? SpiderTheme.brandOrange : SpiderTheme.darkBorder,
              width: uploaded ? 1.5 : 1.0,
            ),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(
                uploaded ? Icons.check_circle : Icons.camera_alt_outlined,
                color: uploaded ? SpiderTheme.brandOrange : SpiderTheme.mutedGray,
                size: 22,
              ),
              const SizedBox(height: 6),
              Text(
                label,
                style: TextStyle(
                  color: uploaded ? SpiderTheme.surfaceLight : SpiderTheme.mutedGray,
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
