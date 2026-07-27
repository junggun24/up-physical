// Package analysis — 분석 엔진 호출 경계.
//
// 운영 핫패스를 Go/Rust로 옮기기 전까지, 검증된 Python 엔진(engine/run_analysis.py)을
// subprocess 로 호출해 재사용한다. 워커는 이 패키지만 의존하므로, 후에 네이티브 구현으로
// 교체할 때 호출부를 바꿀 필요가 없다(인터페이스 안정).
package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Runner struct {
	Python string // 예: "python3"
	Script string // 예: "/app/engine/run_analysis.py"
}

type SubjectResult struct {
	SubjectKey     string          `json:"subject_key"`
	OverallScore   float64         `json:"overall_score"`
	DTWDistance    float64         `json:"dtw_distance"`
	ScoreBreakdown json.RawMessage `json:"score_breakdown"`
	Feedback       json.RawMessage `json:"feedback"`
}

type Result struct {
	Results    []SubjectResult `json:"results"`
	Comparison json.RawMessage `json:"comparison"` // 코치 나란히 비교용 2D(있으면)
}

// Analyze — 사용자/레퍼런스 스트림 원시 JSON을 받아 엔진을 실행, 결과를 반환.
func (r Runner) Analyze(ctx context.Context, userJSON, refJSON []byte) (*Result, error) {
	dir, err := os.MkdirTemp("", "upx-analyze-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	userPath := filepath.Join(dir, "user.json")
	refPath := filepath.Join(dir, "ref.json")
	if err := os.WriteFile(userPath, userJSON, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(refPath, refJSON, 0o600); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, r.Python, r.Script, userPath, refPath)
	var stdout, stderr []byte
	stdout, err = cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
		return nil, fmt.Errorf("엔진 실행 실패: %w (stderr: %s)", err, string(stderr))
	}

	var res Result
	if err := json.Unmarshal(stdout, &res); err != nil {
		return nil, fmt.Errorf("엔진 출력 파싱 실패: %w", err)
	}
	return &res, nil
}
