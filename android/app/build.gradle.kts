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
        versionCode = 1
        // The workflow overrides it with the release tag when there is one.
        versionName = System.getenv("CODDY_VERSION") ?: "0.0.0"
        ndk {
            // The Go build produces exactly these two ABIs (make android).
            abiFilters += listOf("arm64-v8a", "x86_64")
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
