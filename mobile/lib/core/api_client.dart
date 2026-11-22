import 'dart:convert';
import 'package:http/http.dart' as http;

class ApiClient {
  static const String baseUrl = 'http://localhost:8080/api/v1';
  static String? authToken;

  // Local read cache map
  static final Map<String, dynamic> _readCache = {};

  static Future<Map<String, String>> _headers() async {
    final headers = {
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (authToken != null) {
      headers['Authorization'] = 'Bearer $authToken';
    }
    return headers;
  }

  /// Get request with fallback to offline read cache
  static Future<dynamic> get(String endpoint, {bool useCache = true}) async {
    final url = Uri.parse('$baseUrl$endpoint');
    try {
      final headers = await _headers();
      final response = await http.get(url, headers: headers).timeout(const Duration(seconds: 5));
      if (response.statusCode >= 200 && response.statusCode < 300) {
        final data = json.decode(response.body);
        _readCache[endpoint] = data;
        return data;
      }
      throw Exception('Server error: ${response.statusCode}');
    } catch (e) {
      if (useCache && _readCache.containsKey(endpoint)) {
        return _readCache[endpoint];
      }
      rethrow;
    }
  }

  /// Post request - STRICTLY ONLINE. No offline holds or payments to prevent double-booking.
  static Future<dynamic> post(String endpoint, Map<String, dynamic> body) async {
    final url = Uri.parse('$baseUrl$endpoint');
    final headers = await _headers();
    final response = await http
        .post(url, headers: headers, body: json.encode(body))
        .timeout(const Duration(seconds: 8));

    if (response.statusCode >= 200 && response.statusCode < 300) {
      return json.decode(response.body);
    } else {
      final err = json.decode(response.body);
      throw Exception(err['error'] ?? 'API error: ${response.statusCode}');
    }
  }
}
