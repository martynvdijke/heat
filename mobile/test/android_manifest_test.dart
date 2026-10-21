import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final manifest =
      File('android/app/src/main/AndroidManifest.xml').readAsStringSync();
  final gradle = File('android/app/build.gradle.kts').readAsStringSync();

  test('manifest grants INTERNET', () {
    expect(manifest, contains('android.permission.INTERNET'));
  });

  test('manifest permits self-hosted cleartext HTTP', () {
    expect(manifest, contains('android:usesCleartextTraffic="true"'));
  });

  test('release build wires key.properties signing with a debug fallback', () {
    expect(gradle, contains('key.properties'));
    expect(gradle, contains('releaseSigningConfig'));
    expect(gradle, contains('signingConfigs.getByName("debug")'));
  });

  test('app icon is generated from the web favicon with branded styling', () {
    expect(manifest, contains('android:label="HEAT"'));
    expect(manifest, contains('android:icon="@mipmap/ic_launcher"'));

    final adaptive = File('android/app/src/main/res/mipmap-anydpi-v26/ic_launcher.xml')
        .readAsStringSync();
    expect(adaptive, contains('@color/ic_launcher_background'));
    expect(adaptive, contains('@drawable/ic_launcher_foreground'));

    final colors = File('android/app/src/main/res/values/colors.xml').readAsStringSync();
    expect(colors, contains('#111111'));

    final mdpi = File('android/app/src/main/res/mipmap-mdpi/ic_launcher.png');
    expect(mdpi.existsSync(), isTrue);
    expect(mdpi.lengthSync(), greaterThan(0));
  });
}
