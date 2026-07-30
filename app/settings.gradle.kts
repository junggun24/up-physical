// 업 피지컬 Android 앱 프로젝트.
// :core-stream — 골격 스트림 생성 코어 (kotlin-jvm, Android SDK 불필요 → 지금 테스트 가능)
// :app        — 카메라·MediaPipe·UI (2단계, Android SDK 설치 후 추가)
plugins {
    id("org.gradle.toolchains.foojay-resolver-convention") version "1.0.0"
}

rootProject.name = "upphysical-app"

include(":core-stream")
