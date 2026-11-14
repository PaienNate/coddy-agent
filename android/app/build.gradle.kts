plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "dev.coddy.app"
    compileSdk = 35

    defaultConfig {
        applicationId = "dev.coddy.app"
        minSdk = 26
        targetSdk = 35
        // The workflow passes the release tag and a numeric code derived from
        // it; a local build falls back to something installable.
        versionName = System.getenv("CODDY_VERSION") ?: "0.0.0"
        versionCode = System.getenv("CODDY_VERSION_CODE")?.toIntOrNull() ?: 1
        ndk {
            // The Go build produces exactly these two ABIs (make android).
            abiFilters += listOf("arm64-v8a", "x86_64")
        }
    }

    // One APK per ABI: the server binary is ~40 MB each, and a phone only ever
    // runs one of them. A universal APK would double the download for nothing.
    splits {
        abi {
            isEnable = true
            reset()
            include("arm64-v8a", "x86_64")
            isUniversalApk = false
        }
    }

    // A release APK must be signed to install. The release workflow decodes a
    // keystore into ANDROID_KEYSTORE_PATH when the repository carries one; a
    // fork without the secret falls back to the debug build and says so.
    val releaseKeystore = System.getenv("ANDROID_KEYSTORE_PATH")
    signingConfigs {
        if (!releaseKeystore.isNullOrBlank()) {
            create("release") {
                storeFile = file(releaseKeystore)
                storePassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
                keyAlias = System.getenv("ANDROID_KEY_ALIAS")
                keyPassword = System.getenv("ANDROID_KEY_PASSWORD")
            }
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            signingConfigs.findByName("release")?.let { signingConfig = it }
        }
    }

    packaging {
        // Both libcoddy.so and libbusybox.so are executed, not loaded, so they
        // must exist as files in the app's native library directory. Without
        // legacy packaging Android maps them from the APK and execve fails.
        jniLibs {
            useLegacyPackaging = true
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

dependencies {
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.appcompat:appcompat:1.7.0")
}
