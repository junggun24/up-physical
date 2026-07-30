package com.upphysical.stream

import java.time.Instant
import java.util.UUID

/** 포즈 추정기 출력 1점 — MediaPipe PoseLandmarker 결과의 어댑터 타깃 (인덱스 = 키포인트 id). */
data class Landmark(
    val x: Double,
    val y: Double,
    val z: Double? = null,
    val visibility: Double,
)

enum class Dimensions(val wire: String) {
    TWO_D("2d"),
    THREE_D("3d"),
}

/**
 * 촬영 세션 1회 → 계약 준수 골격 스트림.
 *
 * 계약 방어를 업로드 전에 수행한다 (서버 INV 거부 = 사용자에겐 촬영 실패):
 * - INV-2: t 가 증가하지 않는 프레임은 드롭하고 센다 (기기 타임스탬프 중복 현실 대응)
 * - INV-4: 랜드마크는 정확히 33개 (아니면 어댑터 버그 → 즉시 예외)
 * - INV-7: 3d 는 z 필수, 2d 는 z 를 버린다
 * - INV-8: frame_count = 실제 프레임 수
 */
class SkeletonStreamBuilder(
    private val fps: Double,
    private val modelVariant: String,
    private val modelVersion: String,
    private val dimensions: Dimensions = Dimensions.TWO_D,
    private val sessionId: String = UUID.randomUUID().toString(),
    private val subjectId: String = "player-1",
    private val label: String? = "user",
    private val clock: () -> Instant = Instant::now,
) {
    init {
        require(fps > 0) { "fps 는 0보다 커야 함: $fps" }
    }

    private val frames = mutableListOf<Frame>()

    var droppedFrames: Int = 0
        private set

    /** @return 프레임이 채택되면 true, (t 비증가로) 드롭되면 false */
    fun addFrame(tSeconds: Double, landmarks: List<Landmark>): Boolean {
        require(landmarks.size == KEYPOINT_COUNT) {
            "blazepose_33 은 랜드마크 ${KEYPOINT_COUNT}개 필요, 받음: ${landmarks.size}"
        }
        if (dimensions == Dimensions.THREE_D) {
            require(landmarks.all { it.z != null }) { "3d 스트림은 모든 랜드마크에 z 필요" }
        }

        val last = frames.lastOrNull()?.t
        if (last != null && tSeconds <= last) {
            droppedFrames++
            return false
        }

        frames += Frame(
            t = tSeconds,
            keypoints = landmarks.mapIndexed { id, lm ->
                Keypoint(
                    id = id,
                    x = lm.x,
                    y = lm.y,
                    z = if (dimensions == Dimensions.THREE_D) lm.z else null,
                    visibility = lm.visibility,
                )
            },
        )
        return true
    }

    fun build(): SkeletonStream {
        check(frames.isNotEmpty()) { "프레임이 없음 — 촬영 실패로 처리해야 함" }
        return SkeletonStream(
            schemaVersion = SCHEMA_VERSION,
            sessionId = sessionId,
            createdAt = clock().toString(),
            capture = Capture(
                source = "on_device",
                model = "blazepose",
                modelVariant = modelVariant,
                modelVersion = modelVersion,
                fps = fps,
                coordinateSpace = "image_normalized",
                dimensions = dimensions.wire,
                keypointTopology = "blazepose_33",
                frameCount = frames.size,
                durationS = frames.last().t - frames.first().t,
            ),
            subjects = listOf(Subject(subjectId = subjectId, label = label, frames = frames.toList())),
        )
    }

    companion object {
        const val KEYPOINT_COUNT = 33
        const val SCHEMA_VERSION = "1.0"
    }
}
