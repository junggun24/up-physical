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

// pipeline/extract_pose.py 산출물(랜드마크) → 계약 준수 골격 스트림
tasks.register<JavaExec>("buildStream") {
    group = "application"
    description = "랜드마크 JSON 을 골격 스트림으로 조립한다"
    mainClass.set("com.upphysical.stream.BuildStreamKt")
    classpath = sourceSets["main"].runtimeClasspath
    // 경로 인자를 리포 루트 기준으로 쓸 수 있게 고정 (기본값은 모듈 디렉토리라 혼란스럽다)
    workingDir = rootProject.projectDir.parentFile
}

tasks.test {
    useJUnitPlatform()
}
