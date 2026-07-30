package com.upphysical.app

import com.upphysical.stream.Landmark
import com.upphysical.stream.SkeletonStream
import com.upphysical.stream.SkeletonStreamBuilder
import kotlin.math.cos
import kotlin.math.sin

/**
 * 스윙 골격 스트림의 공급원.
 *
 * 다음 단계에서 CameraX + MediaPipe PoseLandmarker 구현체가 이 인터페이스를 채운다
 * (프레임마다 landmarks → [SkeletonStreamBuilder.addFrame]). 지금은 합성 스윙으로
 * 업로드→분석→결과 루프를 실기기에서 검증한다.
 */
interface SwingSource {
    fun capture(): SkeletonStream
}

/** 합성 스윙 — 카메라 없이 파이프라인을 확인하기 위한 개발용 소스. */
class SyntheticSwingSource(
    private val frameCount: Int = 30,
    private val fps: Double = 30.0,
) : SwingSource {

    override fun capture(): SkeletonStream {
        val builder = SkeletonStreamBuilder(
            fps = fps,
            modelVariant = "full",
            modelVersion = "synthetic-0.1",
            // sessionId 기본값 = 새 UUID (세션마다 새로 발급)
        )
        repeat(frameCount) { i ->
            val t = i / fps
            builder.addFrame(t, syntheticLandmarks(t))
        }
        return builder.build()
    }

    private fun syntheticLandmarks(t: Double): List<Landmark> =
        (0 until SkeletonStreamBuilder.KEYPOINT_COUNT).map { k ->
            val phase = t * 2.0 * Math.PI + k * 0.1
            Landmark(
                x = 0.5 + 0.2 * sin(phase),
                y = 0.5 + 0.2 * cos(phase),
                visibility = 0.95,
            )
        }
}
