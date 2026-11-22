class ContainerRoute {
  final String id;
  final String isoCode;
  final String transportMode;
  final String originTitle;
  final String destTitle;
  final DateTime departureTime;
  final double totalCbm;
  final double availableCbm;
  final double totalWeightKg;
  final double availableWeightKg;
  final int ratePerCbmPaise;

  ContainerRoute({
    required this.id,
    required this.isoCode,
    required this.transportMode,
    required this.originTitle,
    required this.destTitle,
    required this.departureTime,
    required this.totalCbm,
    required this.availableCbm,
    required this.totalWeightKg,
    required this.availableWeightKg,
    required this.ratePerCbmPaise,
  });

  double get ratePerCbmRupees => ratePerCbmPaise / 100.0;

  factory ContainerRoute.fromJson(Map<String, dynamic> json) {
    return ContainerRoute(
      id: json['id'] ?? '',
      isoCode: json['iso_container_code'] ?? '',
      transportMode: json['transport_mode'] ?? 'rail',
      originTitle: json['origin_title'] ?? '',
      destTitle: json['dest_title'] ?? '',
      departureTime: DateTime.tryParse(json['departure_time'] ?? '') ?? DateTime.now(),
      totalCbm: (json['total_cbm'] as num?)?.toDouble() ?? 30.0,
      availableCbm: (json['available_cbm'] as num?)?.toDouble() ?? 18.6,
      totalWeightKg: (json['total_weight_kg'] as num?)?.toDouble() ?? 20000.0,
      availableWeightKg: (json['available_weight_kg'] as num?)?.toDouble() ?? 14000.0,
      ratePerCbmPaise: json['rate_per_cbm_paise'] ?? 74000,
    );
  }
}

class ConsignmentQuote {
  final double computedVolumeCbm;
  final double chargeableVolumeCbm;
  final double actualWeightKg;
  final double chargeableWeightKg;
  final int ratePerCbmPaise;
  final int baseAmountPaise;
  final int platformFeePaise;
  final int gstAmountPaise;
  final int totalAmountPaise;

  ConsignmentQuote({
    required this.computedVolumeCbm,
    required this.chargeableVolumeCbm,
    required this.actualWeightKg,
    required this.chargeableWeightKg,
    required this.ratePerCbmPaise,
    required this.baseAmountPaise,
    required this.platformFeePaise,
    required this.gstAmountPaise,
    required this.totalAmountPaise,
  });

  double get totalRupees => totalAmountPaise / 100.0;
  double get baseRupees => baseAmountPaise / 100.0;
  double get rateRupees => ratePerCbmPaise / 100.0;

  factory ConsignmentQuote.fromJson(Map<String, dynamic> json) {
    return ConsignmentQuote(
      computedVolumeCbm: (json['computed_volume_cbm'] as num?)?.toDouble() ?? 0.0,
      chargeableVolumeCbm: (json['chargeable_volume_cbm'] as num?)?.toDouble() ?? 0.0,
      actualWeightKg: (json['actual_weight_kg'] as num?)?.toDouble() ?? 0.0,
      chargeableWeightKg: (json['chargeable_weight_kg'] as num?)?.toDouble() ?? 0.0,
      ratePerCbmPaise: json['rate_per_cbm_paise'] ?? 0,
      baseAmountPaise: json['base_amount_paise'] ?? 0,
      platformFeePaise: json['platform_fee_paise'] ?? 0,
      gstAmountPaise: json['gst_amount_paise'] ?? 0,
      totalAmountPaise: json['total_amount_paise'] ?? 0,
    );
  }
}
