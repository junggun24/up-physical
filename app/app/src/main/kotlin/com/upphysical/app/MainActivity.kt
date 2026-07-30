package com.upphysical.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.tooling.preview.Preview
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContent {
            MaterialTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    AnalysisScreen()
                }
            }
        }
    }
}

@Composable
fun AnalysisScreen(vm: MainViewModel = viewModel()) {
    val state by vm.state.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text("업 피지컬", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        Text("포핸드 분석 (개발 빌드)", style = MaterialTheme.typography.bodyMedium)
        Spacer(Modifier.height(32.dp))

        when (val s = state) {
            is AnalysisUiState.Idle -> IdleView(onStart = vm::startAnalysis)
            is AnalysisUiState.Working -> WorkingView(status = s.status)
            is AnalysisUiState.Success -> ResultView(results = s.results, onAgain = vm::reset)
            is AnalysisUiState.Failed -> FailedView(message = s.message, onRetry = vm::startAnalysis)
        }
    }
}

@Composable
private fun IdleView(onStart: () -> Unit) {
    Text(
        "합성 스윙으로 서버 분석 루프를 확인합니다.\n(카메라·MediaPipe 연결은 다음 단계)",
        style = MaterialTheme.typography.bodyMedium,
    )
    Spacer(Modifier.height(16.dp))
    Button(onClick = onStart) { Text("분석 시작") }
}

@Composable
private fun WorkingView(status: String) {
    CircularProgressIndicator()
    Spacer(Modifier.height(16.dp))
    Text(status, style = MaterialTheme.typography.bodyLarge)
    Spacer(Modifier.height(8.dp))
    Text("분석은 서버에서 진행됩니다", style = MaterialTheme.typography.bodySmall)
}

@Composable
private fun ResultView(results: List<AnalysisResult>, onAgain: () -> Unit) {
    if (results.isEmpty()) {
        Text("결과가 비어 있습니다", style = MaterialTheme.typography.bodyLarge)
    } else {
        val r = results.first()
        Text("종합 점수", style = MaterialTheme.typography.bodyMedium)
        Text(
            "%.1f".format(r.overallScore),
            style = MaterialTheme.typography.displaySmall,
            fontWeight = FontWeight.Bold,
        )
        Spacer(Modifier.height(24.dp))

        // 핵심 원칙: 교정 포인트는 우선순위 1개만 보여준다.
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(Modifier.padding(16.dp)) {
                Text("이번에 고칠 단 하나", fontWeight = FontWeight.Bold)
                Spacer(Modifier.height(8.dp))
                Text(r.topFix ?: "교정 포인트를 찾지 못했습니다")
                r.topFixSegment?.let {
                    Spacer(Modifier.height(4.dp))
                    Text("구간: $it", style = MaterialTheme.typography.bodySmall)
                }
            }
        }
    }
    Spacer(Modifier.height(24.dp))
    OutlinedButton(onClick = onAgain) { Text("다시 하기") }
}

@Composable
private fun FailedView(message: String, onRetry: () -> Unit) {
    Text("분석 실패", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
    Spacer(Modifier.height(8.dp))
    Text(message, style = MaterialTheme.typography.bodyMedium)
    Spacer(Modifier.height(16.dp))
    Button(onClick = onRetry) { Text("다시 시도") }
}

@Preview(showBackground = true)
@Composable
private fun ResultPreview() {
    MaterialTheme {
        Column {
            ResultView(
                results = listOf(
                    AnalysisResult("player-1", 88.5, "테이크백에서 손목을 더 세우세요", "backswing"),
                ),
                onAgain = {},
            )
        }
    }
}
