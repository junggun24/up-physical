package com.upphysical.app

import com.upphysical.stream.SkeletonStream
import java.util.UUID
import java.util.concurrent.TimeUnit
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.jsonArray
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import kotlinx.serialization.json.put
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

/** 분석 결과 중 앱 화면이 쓰는 부분. "고칠 단 하나"는 priority 최솟값 피드백. */
data class AnalysisResult(
    val subjectKey: String,
    val overallScore: Double,
    val topFix: String?,
    val topFixSegment: String?,
)

class ApiException(message: String) : Exception(message)

/**
 * 인제스션 API 클라이언트 — 업로드(멱등) → 잡 폴링 → 결과.
 *
 * 서버 계약: POST /v1/sessions 는 Idempotency-Key 필수, 202 로 job_id 반환.
 * 같은 키 재시도는 같은 잡을 돌려주므로 네트워크 재시도가 중복 분석을 만들지 않는다.
 */
class AnalysisClient(
    private val baseUrl: String = BuildConfig.API_BASE_URL,
    private val userId: String = "app-dev-1",
) {
    private val http = OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .build()

    private val json = Json { ignoreUnknownKeys = true }

    /** 업로드 → 완료까지 폴링 → 결과. onStatus 로 진행 상태를 UI에 알린다. */
    suspend fun analyze(
        stream: SkeletonStream,
        sport: String = "tennis",
        action: String = "forehand",
        onStatus: (String) -> Unit = {},
    ): List<AnalysisResult> = withContext(Dispatchers.IO) {
        onStatus("업로드 중")
        val jobId = createSession(stream, sport, action)

        onStatus("분석 중")
        pollUntilDone(jobId, onStatus)

        onStatus("결과 받는 중")
        fetchResults(jobId)
    }

    private fun createSession(stream: SkeletonStream, sport: String, action: String): String {
        val body = buildJsonObject {
            put("stream", json.parseToJsonElement(stream.toJson()))
            put(
                "analysis",
                buildJsonObject {
                    put("sport", sport)
                    put("action", action)
                },
            )
        }
        val req = Request.Builder()
            .url("$baseUrl/v1/sessions")
            // 멱등키: 이 업로드 시도를 식별. 재시도 시 같은 키를 재사용해야 중복 분석이 없다.
            .header("Idempotency-Key", UUID.randomUUID().toString())
            .header("X-User-Id", userId)
            .post(body.toString().toRequestBody(JSON_MEDIA))
            .build()

        val obj = execute(req, "업로드 실패")
        return obj["job_id"]?.jsonPrimitive?.content
            ?: throw ApiException("응답에 job_id 없음: $obj")
    }

    private suspend fun pollUntilDone(jobId: String, onStatus: (String) -> Unit) {
        repeat(MAX_POLLS) { attempt ->
            val req = Request.Builder().url("$baseUrl/v1/jobs/$jobId").get().build()
            when (val status = execute(req, "잡 조회 실패")["status"]?.jsonPrimitive?.content) {
                "succeeded" -> return
                "failed" -> throw ApiException("분석 실패 — 다시 촬영해 주세요")
                else -> onStatus("분석 중 (${attempt + 1}s, $status)")
            }
            delay(POLL_INTERVAL_MS)
        }
        throw ApiException("분석이 너무 오래 걸립니다 — 잠시 후 다시 확인해 주세요")
    }

    private fun fetchResults(jobId: String): List<AnalysisResult> {
        val req = Request.Builder().url("$baseUrl/v1/jobs/$jobId/results").get().build()
        val results = execute(req, "결과 조회 실패")["results"]?.jsonArray ?: return emptyList()
        return results.map { it.jsonObject.toAnalysisResult() }
    }

    private fun JsonObject.toAnalysisResult(): AnalysisResult {
        // 피드백은 priority 오름차순 중 첫 번째만 노출한다 ("이번에 고칠 단 하나").
        val top = this["feedback"]?.jsonArray
            ?.mapNotNull { it.jsonObject }
            ?.minByOrNull { it["priority"]?.jsonPrimitive?.content?.toIntOrNull() ?: Int.MAX_VALUE }
        return AnalysisResult(
            subjectKey = this["subject_key"]?.jsonPrimitive?.content ?: "?",
            overallScore = this["overall_score"]?.jsonPrimitive?.content?.toDoubleOrNull() ?: 0.0,
            topFix = top?.get("message")?.jsonPrimitive?.content,
            topFixSegment = top?.get("segment")?.jsonPrimitive?.content,
        )
    }

    private fun execute(req: Request, failMessage: String): JsonObject {
        http.newCall(req).execute().use { res ->
            val text = res.body?.string().orEmpty()
            if (!res.isSuccessful) {
                // 서버는 problem+json 으로 detail 을 준다. 사용자 메시지는 간결하게.
                val detail = runCatching {
                    json.parseToJsonElement(text).jsonObject["detail"]?.jsonPrimitive?.content
                }.getOrNull()
                throw ApiException("$failMessage (${res.code})${detail?.let { ": $it" } ?: ""}")
            }
            return json.parseToJsonElement(text).jsonObject
        }
    }

    companion object {
        private val JSON_MEDIA = "application/json; charset=utf-8".toMediaType()
        private const val POLL_INTERVAL_MS = 1_000L
        private const val MAX_POLLS = 60
    }
}
