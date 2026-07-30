package com.upphysical.stream

import java.io.File
import java.time.Instant
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * pipeline/extract_pose.py 가 뽑은 랜드마크 시계열 → 계약 준수 골격 스트림.
 *
 * 계약 조립기를 한 벌로 유지하기 위해 앱과 같은 [SkeletonStreamBuilder] 를 쓴다
 * (Python 쪽에 계약 구현체를 만들지 않는다 — 드리프트 방지).
 *
 * 사용: ./gradlew :core-stream:buildStream --args="landmarks.json out.json [session-id]"
 */

@Serializable
private data class ExtractMeta(
    val fps: Double,
    @SerialName("model_variant") val modelVariant: String = "full",
    @SerialName("source_file") val sourceFile: String = "",
)

@Serializable
private data class ExtractLandmark(
    val x: Double,
    val y: Double,
    val z: Double? = null,
    val visibility: Double,
)

@Serializable
private data class ExtractFrame(val t: Double, val landmarks: List<ExtractLandmark>)

@Serializable
private data class ExtractFile(val meta: ExtractMeta, val frames: List<ExtractFrame>)

fun main(args: Array<String>) {
    if (args.size < 2) {
        System.err.println("사용: buildStream <landmarks.json> <out.json> [session-id]")
        kotlin.system.exitProcess(2)
    }
    val input = File(args[0])
    val output = File(args[1])

    val parsed = Json { ignoreUnknownKeys = true }
        .decodeFromString(ExtractFile.serializer(), input.readText())

    val builder = SkeletonStreamBuilder(
        fps = parsed.meta.fps,
        modelVariant = parsed.meta.modelVariant,
        modelVersion = "mediapipe-1.0",
        // 2d 로 고정: image_normalized 공간의 z 는 신뢰도가 낮아 계약상 생략한다(INV-7 회피).
        dimensions = Dimensions.TWO_D,
        sessionId = args.getOrNull(2) ?: java.util.UUID.randomUUID().toString(),
        subjectId = "coach-1",
        label = "coach",
        clock = { Instant.now() },
    )

    var accepted = 0
    parsed.frames.forEach { f ->
        val ok = builder.addFrame(
            f.t,
            f.landmarks.map { Landmark(x = it.x, y = it.y, z = it.z, visibility = it.visibility) },
        )
        if (ok) accepted++
    }

    val stream = builder.build()
    output.writeText(stream.toJson() + "\n")

    println("■ ${input.name} → ${output.name}")
    println("  입력 프레임 ${parsed.frames.size} · 채택 $accepted · t 비증가 드롭 ${builder.droppedFrames}")
    println("  fps ${parsed.meta.fps} · frame_count ${stream.capture.frameCount} · duration ${stream.capture.durationS}s")
    println("  session_id ${stream.sessionId}")
}
