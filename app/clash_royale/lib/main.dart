// File: lib/main.dart
import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';
import 'services/auth_service.dart';
import 'screens/login_page.dart';
import 'screens/home_page.dart';

void main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(fileName: ".env");

  final auth = AuthService();
  final loggedIn = await auth.isLoggedIn();

  runApp(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      home: loggedIn ? HomePage() : LoginPage(),
    ),
  );
}
