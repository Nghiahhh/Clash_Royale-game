import 'package:clash_royale/Clash_Royale.dart';
import 'package:flame/flame.dart';
import 'package:flame/game.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  Flame.device.fullScreen();
  Flame.device.setPortrait();

  Clash_Royale game = Clash_Royale();

  // runApp(
  //   MaterialApp(
  //     debugShowCheckedModeBanner: false,
  //     home: GameWidget(game: game),
  //   ),
  // );
  runApp(
    MaterialApp(
      debugShowCheckedModeBanner: false,
      home: GameWidget(game: kDebugMode ? Clash_Royale() : game),
    ),
  );
}
