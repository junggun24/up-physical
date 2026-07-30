plugins {
    id("com.android.application") version "8.13.0"
    kotlin("android") version "2.2.20"
    kotlin("plugin.compose") version "2.2.20"
    kotlin("plugin.serialization") version "2.2.20"
}

android {
    namespace = "com.upphysical.app"
    compileSdk = 36

    defaultConfig {
        applicationId = "com.upphysical.app"
        minSdk = 26
        targetSdk = 36
        versionCode = 1
        versionName = "0.1.0"

        // 백엔드 주소: 에뮬레이터에서 호스트 머신은 10.0.2.2.
        // 실기기는 같은 와이파이의 맥 IP로 바꾼다 (RUN.md 참조).
        buildConfigField("String", "API_BASE_URL", "\"http://10.0.2.2:8080\"")
    }

    buildFeatures {
        compose = true
        buildConfig = true
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }

    compileOptions {
        // Kotlin jvmToolchain(17) 과 일치해야 한다 (AGP 가 불일치를 에러로 잡는다).
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlin {
        jvmToolchain(17)
    }
}

dependencies {
    implementation(project(":core-stream"))

    implementation(platform("androidx.compose:compose-bom:2025.09.00"))
    implementation("androidx.compose.ui:ui")
    implementation("androidx.compose.material3:material3")
    implementation("androidx.compose.ui:ui-tooling-preview")
    debugImplementation("androidx.compose.ui:ui-tooling")

    implementation("androidx.activity:activity-compose:1.11.0")
    implementation("androidx.lifecycle:lifecycle-viewmodel-compose:2.9.4")
    implementation("androidx.core:core-ktx:1.17.0")

    implementation("com.squareup.okhttp3:okhttp:5.1.0")
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-android:1.10.2")

    testImplementation(kotlin("test"))
}
