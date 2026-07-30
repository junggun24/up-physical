plugins {
    kotlin("jvm") version "2.2.20"
    kotlin("plugin.serialization") version "2.2.20"
    application
}

repositories {
    mavenCentral()
}

kotlin {
    // Android 모듈과 맞출 안정 타깃. 로컬에 없으면 foojay 리졸버가 자동 프로비저닝.
    jvmToolchain(17)
}

dependencies {
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")
    testImplementation(kotlin("test"))
}

application {
    // 계약 교차 검증용 샘플 스트림 생성기 (.harness/runners/check-app.sh 가 사용)
    mainClass.set("com.upphysical.stream.GenerateSampleKt")
}

tasks.test {
    useJUnitPlatform()
}
