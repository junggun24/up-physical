---
problem: Android 앱 모듈 빌드가 Gradle/AGP/JDK/테마 4중 비호환으로 연속 실패
module: app
symptoms: ["relies on 'org.gradle.api.problems.internal.InternalProblems', a Gradle internal API that was removed in Gradle 9.6.0", "Server returned HTTP response code: 404 for URL: .../gradle-9.5-bin.zip", "resource style/Theme.Material3.DayNight.NoActionBar not found", "Inconsistent JVM-target compatibility detected for tasks 'compileDebugJavaWithJavac' (11) and 'compileDebugKotlin' (17)"]
root_cause: 최신 Gradle이 AGP 8.x가 의존하는 내부 API를 제거했고, 나머지는 버전·리소스·타깃 불일치
tags: [android, gradle, agp, jdk, compose, theme, jvm-target]
date: 2026-07-30
---

## 증상

`:app:assembleDebug` 가 네 단계에 걸쳐 순차 실패 (각 수정 후 다음 에러 노출).

## 근본 원인과 해결 (4건)

1. **Gradle 9.6.1 ↔ AGP 8.13 비호환** — Gradle 9.6.0이 `InternalProblems` 내부 API를 제거.
   → 래퍼를 **9.5.1** 로 고정 (에러 메시지가 9.5 사용을 직접 권고).
2. **`gradle-9.5-bin.zip` 404** — 그런 버전명이 없다. `services.gradle.org/versions/all` 로
   실제 목록 확인 → **9.5.1** 이 정확한 이름. (버전명을 추측하지 말고 조회한다.)
3. **`Theme.Material3.*` 리소스 없음** — Compose Material3는 `com.google.android.material`
   XML 테마를 제공하지 않는다. → 플랫폼 테마(`android:Theme.DeviceDefault.DayNight`) 기반
   자체 스타일 정의. `.DayNight.NoActionBar` 조합은 플랫폼에 없어 액션바는 item으로 끈다.
4. **JVM 타깃 불일치** — `compileOptions` 11 vs Kotlin `jvmToolchain(17)`.
   → 둘 다 17로 통일.

추가: JDK 24/26은 AGP 미지원 → `org.gradle.java.home` 을 Android Studio 번들 **JBR 21** 로 고정.

## 재발 방지

- 확정 조합을 `RUN.md` 에 명시: Gradle 9.5.1 + AGP 8.13 + Kotlin 2.2.20 + JBR 21 + compileSdk 36.
- 버전 올릴 때는 한 번에 하나만 올리고 `:app:assembleDebug` 로 확인.
- 교훈: 툴체인 에러는 대개 **다음 에러를 가린다** — 추측으로 여러 개를 동시에 바꾸지 말고
  한 번에 하나씩 고치며 에러가 알려주는 대로 따라간다.
