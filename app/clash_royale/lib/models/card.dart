// lib/models/card.dart
class CardModel {
  final String id;
  final String name;
  final int power;

  CardModel({required this.id, required this.name, required this.power});

  factory CardModel.fromJson(Map<String, dynamic> json) {
    return CardModel(id: json['id'], name: json['name'], power: json['power']);
  }
}
