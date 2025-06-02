// lib/services/game_service.dart
import 'package:flutter/material.dart';
import 'dart:convert';
import 'package:http/http.dart' as http;
import '../env/env.dart';
import '../models/card.dart';

class GameService {
  final String baseUrl = dotenv.env['API_URL'] ?? 'http://localhost:3000';

  Future<List<Card>> getCards() async {
    try {
      final response = await http.get(Uri.parse('$baseUrl/cards'));

      if (response.statusCode == 200) {
        final List<dynamic> data = json.decode(response.body);
        return data.map((json) => Card.fromJson(json)).toList();
      } else {
        throw Exception('Failed to load cards');
      }
    } catch (e) {
      throw Exception('Error connecting to server: $e');
    }
  }

  // Thêm các phương thức game khác ở đây
}
