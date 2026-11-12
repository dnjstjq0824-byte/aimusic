import java.io.FileInputStream
import java.util.Properties

plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

val keystoreProperties = Properties().apply {
    val f = rootProject.file("key.properties")
    if (f.exists()) load(FileInputStream(f))
}

android {
    namespace = "com.avibe.avibe"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = flutter.ndkVersion

    signingConfigs {
        // Release keystore (see docs/PROD_RUNBOOK.md "Android signing").
        // key.properties is git-ignored; CI injects it from secrets.
        create("release") {
            keyAlias = keystoreProperties["keyAlias"] as String?
            keyPassword = keystoreProperties["keyPassword"] as String?
            storeFile = (keystoreProperties["storeFile"] as String?)?.let { rootProject.file(it) }
            storePassword = keystoreProperties["storePassword"] as String?
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "com.avibe.avibe"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = flutter.minSdkVersion
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    buildTypes {
        release {
            // F-07: release builds always use the release keystore. A missing
            // key fails the build (guard below) instead of silently signing
            // with debug keys.
            signingConfig = signingConfigs.getByName("release")
        }
    }
}

// F-07: fail release assembly/bundle early when the release key is missing.
// Checked against the task graph so debug builds and IDE syncs are unaffected.
gradle.taskGraph.whenReady {
    val releaseBuild = allTasks.any {
        (it.name.startsWith("assemble") || it.name.startsWith("bundle")) &&
            it.name.contains("Release")
    }
    if (releaseBuild && (keystoreProperties["storeFile"] as String?).isNullOrEmpty()) {
        throw GradleException(
            "Release signing key missing: create android/key.properties " +
            "(storeFile/keyAlias/keyPassword/storePassword); " +
            "see docs/PROD_RUNBOOK.md \"Android signing\". " +
            "Refusing to sign a release build with debug keys."
        )
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}
