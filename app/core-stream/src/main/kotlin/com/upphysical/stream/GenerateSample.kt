package com.upphysical.stream

import java.io.File
import java.time.Instant
import kotlin.math.cos
import kotlin.math.sin

/**
 * 계약 교차 검증용 샘플 생성기 — 이 모듈이 만든 스트림을 Go 검증기(cmd/validate)에 통과시킨다.
 * (.harness/runners/check-app.sh) 결정적 출력: 같은 코드는 같은 파일을 만든다.
 */
fun main(args: Array<String>) {
    val builder = SkeletonStreamBuilder(
        fps = 30.0,
        modelVariant = "full",
        modelVersion = "0.10.0",
        sessionId = "33333333-3333-4333-8333-333333333333",
        clock = { Instant.parse("2026-07-30T00:00:00Z") },
    )
    repeat(10) { i ->
        val t = i / 30.0
        builder.addFrame(
            t,
            (0 until 33).map { k ->
                val phase = t * 2.0 * Math.PI + k * 0.1
                Landmark(x = 0.5 + 0.2 * sin(phase), y = 0.5 + 0.2 * cos(phase), visibility = 0.95)
            },
        )
    }
    val json = builder.build().toJson()
    if (args.isEmpty()) println(json) else File(args[0]).writeText(json + "\n")
}
