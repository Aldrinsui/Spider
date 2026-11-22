import 'package:flutter/material.dart';
import '../core/theme.dart';

class AdminReviewScreen extends StatefulWidget {
  const AdminReviewScreen({super.key});

  @override
  State<AdminReviewScreen> createState() => _AdminReviewScreenState();
}

class _AdminReviewScreenState extends State<AdminReviewScreen> {
  int queueCount = 3;
  String currentStatus = 'pending';

  void _handleDecision(bool approved) {
    setState(() {
      currentStatus = approved ? 'approved' : 'rejected';
      if (queueCount > 0) queueCount--;
    });

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(approved
            ? 'Provider Approved! Role updated & audit log recorded.'
            : 'Provider Rejected. Audit log recorded.'),
        backgroundColor: approved ? SpiderTheme.successGreen : SpiderTheme.dangerRed,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: SpiderTheme.darkBackground,
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back, color: SpiderTheme.surfaceLight),
          onPressed: () => Navigator.of(context).pop(),
        ),
        title: const Text('Admin Console'),
      ),
      body: SafeArea(
        child: Column(
          children: [
            // Top Bar
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 12),
              decoration: const BoxDecoration(
                border: Border(
                  bottom: BorderSide(color: SpiderTheme.darkBorder, width: 1),
                ),
              ),
              child: Row(
                children: [
                  const Text(
                    'Cargo Space Admin',
                    style: TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontWeight: FontWeight.bold,
                      fontSize: 15,
                    ),
                  ),
                  const SizedBox(width: 12),
                  const Text(
                    'Provider queue',
                    style: TextStyle(
                      color: SpiderTheme.brandOrange,
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    '$queueCount waiting',
                    style: const TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontFamily: 'IBM Plex Mono',
                      fontSize: 13,
                    ),
                  ),
                ],
              ),
            ),

            // Main Content Area
            Expanded(
              child: ListView(
                padding: const EdgeInsets.all(18),
                children: [
                  // Documents Section
                  const Text(
                    'DOCUMENTS · RAIL LOGISTICS CO.',
                    style: TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontSize: 12,
                      letterSpacing: 0.8,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 12),
                  GridView.count(
                    crossAxisCount: 2,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    mainAxisSpacing: 10,
                    crossAxisSpacing: 10,
                    childAspectRatio: 2.2,
                    children: [
                      _buildDocCard('Transport permit', Icons.description),
                      _buildDocCard('Fleet RC', Icons.directions_bus),
                      _buildDocCard('Container front', Icons.photo_camera),
                      _buildDocCard('Container doors', Icons.door_sliding_outlined),
                    ],
                  ),

                  const SizedBox(height: 24),
                  const Divider(color: SpiderTheme.darkBorder, height: 1),
                  const SizedBox(height: 18),

                  // Decision Details
                  const Text(
                    'DECISION',
                    style: TextStyle(
                      color: SpiderTheme.brandOrange,
                      fontSize: 12,
                      letterSpacing: 0.8,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 6),
                  const Text(
                    'Rail Logistics Co.',
                    style: TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontSize: 22,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  const SizedBox(height: 4),
                  const Text(
                    'MSKU 482913 · GSTIN verified (33AAAAA0000A1Z5)',
                    style: TextStyle(
                      color: SpiderTheme.mutedGray,
                      fontFamily: 'IBM Plex Mono',
                      fontSize: 13,
                    ),
                  ),
                  const SizedBox(height: 12),
                  const Divider(color: SpiderTheme.darkBorder, height: 1),
                  const SizedBox(height: 12),

                  const Text(
                    'Permit number matches the RC. Container photos are geotagged with GPS coordinates (13.0827° N, 80.2707° E).',
                    style: TextStyle(
                      color: SpiderTheme.surfaceLight,
                      fontSize: 14,
                      height: 1.4,
                    ),
                  ),

                  const SizedBox(height: 32),

                  // Actions: Reject / Approve
                  if (currentStatus == 'pending')
                    Row(
                      children: [
                        Expanded(
                          child: OutlinedButton(
                            style: OutlinedButton.styleFrom(
                              foregroundColor: SpiderTheme.dangerRed,
                              side: const BorderSide(color: SpiderTheme.dangerRed),
                              padding: const EdgeInsets.symmetric(vertical: 16),
                              shape: const RoundedRectangleBorder(borderRadius: BorderRadius.zero),
                            ),
                            onPressed: () => _handleDecision(false),
                            child: const Text('REJECT'),
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          flex: 2,
                          child: ElevatedButton(
                            style: ElevatedButton.styleFrom(
                              backgroundColor: SpiderTheme.brandOrange,
                              foregroundColor: SpiderTheme.darkBackground,
                              padding: const EdgeInsets.symmetric(vertical: 16),
                            ),
                            onPressed: () => _handleDecision(true),
                            child: const Text('APPROVE PROVIDER'),
                          ),
                        ),
                      ],
                    )
                  else
                    Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(
                        color: currentStatus == 'approved'
                            ? SpiderTheme.successGreen.withOpacity(0.15)
                            : SpiderTheme.dangerRed.withOpacity(0.15),
                        border: Border.all(
                          color: currentStatus == 'approved'
                              ? SpiderTheme.successGreen
                              : SpiderTheme.dangerRed,
                        ),
                      ),
                      child: Text(
                        currentStatus == 'approved'
                            ? 'PROVIDER APPROVED & AUDIT LOG RECORDED'
                            : 'PROVIDER REJECTED & AUDIT LOG RECORDED',
                        style: TextStyle(
                          color: currentStatus == 'approved'
                              ? SpiderTheme.successGreen
                              : SpiderTheme.dangerRed,
                          fontFamily: 'IBM Plex Mono',
                          fontWeight: FontWeight.bold,
                          fontSize: 13,
                        ),
                        textAlign: TextAlign.center,
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

  Widget _buildDocCard(String label, IconData icon) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: const Color(0xFF1B1F24),
        border: Border.all(color: SpiderTheme.darkBorder),
      ),
      child: Row(
        children: [
          Icon(icon, color: SpiderTheme.brandOrange, size: 20),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              label,
              style: const TextStyle(
                color: SpiderTheme.surfaceLight,
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      ),
    );
  }
}
