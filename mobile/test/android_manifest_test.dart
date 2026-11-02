import 'dart:io';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final manifest =
      File('android/app/src/main/AndroidManifest.xml').readAsStringSync();
  final debugManifest =
      File('android/app/src/debug/AndroidManifest.xml').readAsStringSync();
  final gradle = File('android/app/build.gradle.kts').readAsStringSync();

  test('manifest grants INTERNET', () {
    expect(manifest, contains('android.permission.INTERNET'));
  });

  test('release manifest refuses cleartext HTTP', () {
    expect(manifest, contains('android:usesCleartextTraffic="false"'));
    expect(debugManifest, contains('android:usesCleartextTraffic="true"'));
    expect(debugManifest, contains('tools:replace="android:usesCleartextTraffic"'));
  });

  test('release manifest disables backups and data extraction', () {
    expect(manifest, contains('android:allowBackup="false"'));
    expect(manifest, contains('android:fullBackupContent="false"'));
    expect(manifest, contains('android:dataExtractionRules="@xml/data_extraction_rules"'));
    final rules = File('android/app/src/main/res/xml/data_extraction_rules.xml')
        .readAsStringSync();
    expect(rules, contains('<cloud-backup>'));
    expect(rules, contains('<device-transfer>'));
    expect(rules, contains('domain="sharedpref"'));
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
