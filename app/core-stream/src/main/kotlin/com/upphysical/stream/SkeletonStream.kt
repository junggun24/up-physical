package com.upphysical.stream

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

/**
 * 골격 스트림 계약 모델 — 단일 진실원 contracts/skeleton-stream 의 와이어 포맷.
 * 서버(internal/contract)와 필드명이 1:1 이어야 한다. 이름 변경은 계약 변경이다.
 */

@Serializable
data class Keypoint(
    val id: Int,
    val x: Double,
    val y: Double,
    val z: Double? = null, // 2d 에서는 생략 (explicitNulls=false)
    val visibility: Double,
)

@Serializable
data class Frame(
    val t: Double,
    val keypoints: List<Keypoint>,
)

@Serializable
data class Subject(
    @SerialName("subject_id") val subjectId: String,
    val label: String? = null,
    val frames: List<Frame>,
)

@Serializable
data class Capture(
    val source: String,
    val model: String,
    @SerialName("model_variant") val modelVariant: String,
    @SerialName("model_version") val modelVersion: String,
    val fps: Double,
    @SerialName("coordinate_space") val coordinateSpace: String,
    val dimensions: String,
    @SerialName("keypoint_topology") val keypointTopology: String,
    @SerialName("frame_count") val frameCount: Int,
    @SerialName("duration_s") val durationS: Double? = null,
)

@Serializable
data class SkeletonStream(
    @SerialName("schema_version") val schemaVersion: String,
    @SerialName("session_id") val sessionId: String,
    @SerialName("created_at") val createdAt: String,
    val capture: Capture,
    val subjects: List<Subject>,
) {
    fun toJson(): String = wire.encodeToString(serializer(), this)

    companion object {
        // 계약 직렬화 설정: null 필드 생략(2d의 z), 기본값 포함.
        internal val wire = Json {
            explicitNulls = false
            encodeDefaults = true
        }
    }
}
