// 업 피지컬 Android 앱 프로젝트.
// :core-stream — 골격 스트림 생성 코어 (kotlin-jvm, SDK 불필요 → 단독 테스트 가능)
// :app         — Android 앱 (Compose UI, 업로드·폴링·결과. 카메라/MediaPipe는 다음 단계)
pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "upphysical-app"

include(":core-stream")
include(":app")
