package com.upphysical.app

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch

/**
 * 분석 루프 화면 상태.
 *
 * 분석은 비동기(업로드→202→폴링)라 대기·실패 상태가 1급 시민이다 (design.md 원칙 2).
 */
sealed interface AnalysisUiState {
    data object Idle : AnalysisUiState
    data class Working(val status: String) : AnalysisUiState
    data class Success(val results: List<AnalysisResult>) : AnalysisUiState
    data class Failed(val message: String) : AnalysisUiState
}

class MainViewModel(
    private val swingSource: SwingSource = SyntheticSwingSource(),
    private val client: AnalysisClient = AnalysisClient(),
) : ViewModel() {

    private val _state = MutableStateFlow<AnalysisUiState>(AnalysisUiState.Idle)
    val state: StateFlow<AnalysisUiState> = _state.asStateFlow()

    fun startAnalysis() {
        if (_state.value is AnalysisUiState.Working) return // 중복 실행 방지
        _state.value = AnalysisUiState.Working("스윙 준비 중")
        viewModelScope.launch {
            runCatching {
                val stream = swingSource.capture()
                client.analyze(stream) { status ->
                    _state.value = AnalysisUiState.Working(status)
                }
            }.onSuccess { results ->
                _state.value = AnalysisUiState.Success(results)
            }.onFailure { e ->
                _state.value = AnalysisUiState.Failed(e.message ?: "알 수 없는 오류")
            }
        }
    }

    fun reset() {
        _state.value = AnalysisUiState.Idle
    }
}
