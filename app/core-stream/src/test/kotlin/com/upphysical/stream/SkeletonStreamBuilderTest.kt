package com.upphysical.stream

import java.time.Instant
import kotlin.test.Test
import kotlin.test.assertContains
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * 계약(contracts/skeleton-stream, 서버 internal/contract INV-1..8)의 앱 쪽 의무 명세.
 * 여기서 통과한 스트림은 check-app.sh 에서 Go 검증기(cmd/validate)로 교차 검증된다.
 */
class SkeletonStreamBuilderTest {

    private val fixedClock = { Instant.parse("2026-07-30T00:00:00Z") }

    private fun landmarks33(offset: Double = 0.0, z: Double? = null): List<Landmark> =
        (0 until 33).map { i ->
            Landmark(x = 0.5 + offset + i * 0.001, y = 0.5 - i * 0.001, z = z, visibility = 0.95)
        }

    private fun builder(dimensions: Dimensions = Dimensions.TWO_D) = SkeletonStreamBuilder(
        fps = 30.0,
        modelVariant = "full",
        modelVersion = "0.10.0",
        dimensions = dimensions,
        sessionId = "33333333-3333-4333-8333-333333333333",
        clock = fixedClock,
    )

    @Test
    fun `유효 스트림 - 33개 키포인트 id 0-32, frame_count 일치`() {
        val b = builder()
        b.addFrame(0.0, landmarks33())
        b.addFrame(1 / 30.0, landmarks33(offset = 0.01))
        val s = b.build()

        assertEquals(2, s.capture.frameCount)                       // INV-8
        assertEquals(2, s.subjects.single().frames.size)
        val ids = s.subjects.single().frames.first().keypoints.map { it.id }
        assertEquals((0 until 33).toList(), ids)                    // INV-4: 0..32 정확히
        assertEquals("blazepose_33", s.capture.keypointTopology)
        assertEquals("player-1", s.subjects.single().subjectId)
    }

    @Test
    fun `랜드마크 개수가 33이 아니면 어댑터 오류로 즉시 실패`() {
        val b = builder()
        assertFailsWith<IllegalArgumentException> { b.addFrame(0.0, landmarks33().dropLast(1)) }
    }

    @Test
    fun `t가 증가하지 않는 프레임은 드롭하고 카운트한다`() {          // INV-2 방어
        val b = builder()
        assertTrue(b.addFrame(0.0, landmarks33()))
        assertFalse(b.addFrame(0.0, landmarks33()))                  // 같은 t → 드롭
        assertFalse(b.addFrame(-1.0, landmarks33()))                 // 되돌아간 t → 드롭
        assertTrue(b.addFrame(0.1, landmarks33()))
        assertEquals(2, b.droppedFrames)
        assertEquals(2, b.build().capture.frameCount)
    }

    @Test
    fun `3d는 z 없는 랜드마크를 거부한다`() {                        // INV-7 방어
        val b = builder(dimensions = Dimensions.THREE_D)
        assertFailsWith<IllegalArgumentException> { b.addFrame(0.0, landmarks33(z = null)) }
        b.addFrame(0.0, landmarks33(z = -0.2))                       // z 있으면 통과
    }

    @Test
    fun `2d는 z를 직렬화에서 생략한다`() {
        val b = builder()
        b.addFrame(0.0, landmarks33(z = -0.2))                       // 입력에 z가 와도
        val json = b.build().toJson()
        assertFalse("\"z\"" in json, "2d 스트림에 z가 직렬화됨: $json")
    }

    @Test
    fun `직렬화 필드명과 고정값이 계약과 일치한다`() {
        val b = builder()
        b.addFrame(0.0, landmarks33())
        val json = b.build().toJson()
        assertContains(json, "\"schema_version\":\"1.0\"")
        assertContains(json, "\"session_id\":\"33333333-3333-4333-8333-333333333333\"")
        assertContains(json, "\"source\":\"on_device\"")
        assertContains(json, "\"model\":\"blazepose\"")
        assertContains(json, "\"coordinate_space\":\"image_normalized\"")
        assertContains(json, "\"dimensions\":\"2d\"")
        assertContains(json, "\"keypoint_topology\":\"blazepose_33\"")
        assertContains(json, "\"frame_count\":1")
        assertContains(json, "\"created_at\":\"2026-07-30T00:00:00Z\"")
        assertContains(json, "\"subject_id\":\"player-1\"")
    }

    @Test
    fun `프레임 없이 build 하면 실패한다`() {
        assertFailsWith<IllegalStateException> { builder().build() }
    }

    @Test
    fun `duration_s 는 첫-끝 t 차이다`() {
        val b = builder()
        b.addFrame(1.0, landmarks33())
        b.addFrame(1.5, landmarks33())
        b.addFrame(2.0, landmarks33())
        assertEquals(1.0, b.build().capture.durationS)
    }

    @Test
    fun `fps 는 양수여야 한다`() {
        assertFailsWith<IllegalArgumentException> {
            SkeletonStreamBuilder(fps = 0.0, modelVariant = "full", modelVersion = "1")
        }
    }

    @Test
    fun `세션 id 는 기본적으로 매번 새로 발급된다`() {               // session_id 재사용 500 방지
        val a = SkeletonStreamBuilder(fps = 30.0, modelVariant = "full", modelVersion = "1")
        val b = SkeletonStreamBuilder(fps = 30.0, modelVariant = "full", modelVersion = "1")
        a.addFrame(0.0, landmarks33()); b.addFrame(0.0, landmarks33())
        assertTrue(a.build().sessionId != b.build().sessionId)
        assertNull(null) // 자리표시 없음 — 위 비교가 본 검증
    }
}
