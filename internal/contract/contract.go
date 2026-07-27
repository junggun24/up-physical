// Package contract — 골격 스트림 계약 타입 + 경계 검증(불변식 INV-1..8).
//
// 단일 진실원: contracts/skeleton-stream/{SPEC.md,schema,validate.py}.
// 인제스션 API의 경계 방어는 계약 검증기(validate.py)와 동일 규칙을 Go로 이식한 것이다.
// (스키마 구조/타입 검증 + 스키마로 표현 불가한 불변식.)
package contract

import (
	"encoding/json"
	"fmt"
)

// ─────────────────────────── 타입 ───────────────────────────

// Keypoint — z 는 2d 에서 생략/널 가능하므로 포인터.
type Keypoint struct {
	ID         int      `json:"id"`
	X          float64  `json:"x"`
	Y          float64  `json:"y"`
	Z          *float64 `json:"z"`
	Visibility float64  `json:"visibility"`
}

type Frame struct {
	T         float64    `json:"t"`
	Keypoints []Keypoint `json:"keypoints"`
}

type Subject struct {
	SubjectID string  `json:"subject_id"`
	Label     *string `json:"label,omitempty"`
	Frames    []Frame `json:"frames"`
}

type Capture struct {
	Source           string   `json:"source"`
	Model            string   `json:"model"`
	ModelVariant     string   `json:"model_variant"`
	ModelVersion     string   `json:"model_version"`
	FPS              float64  `json:"fps"`
	CoordinateSpace  string   `json:"coordinate_space"`
	Dimensions       string   `json:"dimensions"`
	KeypointTopology string   `json:"keypoint_topology,omitempty"`
	FrameCount       *int     `json:"frame_count,omitempty"`
	DurationS        *float64 `json:"duration_s,omitempty"`
	VariableFramerate bool    `json:"variable_framerate,omitempty"`
	ZSignConvention  string   `json:"z_sign_convention,omitempty"`
}

type Stream struct {
	SchemaVersion string    `json:"schema_version"`
	SessionID     string    `json:"session_id"`
	CreatedAt     string    `json:"created_at,omitempty"`
	Capture       Capture   `json:"capture"`
	Subjects      []Subject `json:"subjects"`
}

// ValErr — 필드 경로 + 메시지.
type ValErr struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e ValErr) Error() string { return e.Path + ": " + e.Message }

var topologyCounts = map[string]int{"blazepose_33": 33, "coco_17": 17}

var (
	validSources = map[string]bool{"on_device": true, "tracking_camera": true}
	validSpaces  = map[string]bool{"image_normalized": true, "world_metric": true, "root_relative": true}
	validDims    = map[string]bool{"2d": true, "3d": true}
)

// Parse — 원시 바이트를 Stream 으로 디코드(엄격: 알 수 없는 필드는 허용하되 타입 오류는 잡음).
func Parse(raw []byte) (*Stream, error) {
	var s Stream
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %w", err)
	}
	return &s, nil
}

// Validate — 구조 필수값 + 불변식(INV-1..8) 검증. 오류 리스트(빈 슬라이스면 통과).
// validate.py 의 check_invariants 와 동일 규칙. INV-5/6 은 경고라 실패시키지 않는다.
func Validate(s *Stream) []ValErr {
	var errs []ValErr
	add := func(path, msg string) { errs = append(errs, ValErr{Path: path, Message: msg}) }

	// ── 구조/열거형(스키마 레벨) ──
	if s.SchemaVersion == "" {
		add("schema_version", "필수값 누락")
	}
	if s.SessionID == "" {
		add("session_id", "필수값 누락")
	}
	cp := s.Capture
	if !validSources[cp.Source] {
		add("capture.source", "허용되지 않은 값: "+cp.Source)
	}
	if !validSpaces[cp.CoordinateSpace] {
		add("capture.coordinate_space", "허용되지 않은 값: "+cp.CoordinateSpace)
	}
	if !validDims[cp.Dimensions] {
		add("capture.dimensions", "허용되지 않은 값: "+cp.Dimensions)
	}
	if cp.FPS <= 0 {
		add("capture.fps", "fps 는 0보다 커야 함")
	}
	if len(s.Subjects) == 0 {
		add("subjects", "최소 1명 필요")
		return errs // 이후 검사 무의미
	}

	// 토폴로지 기대 개수
	topo := cp.KeypointTopology
	if topo == "" && cp.Model == "blazepose" {
		topo = "blazepose_33"
	}
	expectedN, hasExpected := topologyCounts[topo]

	// ── 불변식 ──
	// INV-1: subject_id 세션 내 유일
	seen := map[string]bool{}
	for _, sub := range s.Subjects {
		if seen[sub.SubjectID] {
			add("invariant", "INV-1 subject_id 중복: "+sub.SubjectID)
		}
		seen[sub.SubjectID] = true
	}

	maxLen := 0
	for _, sub := range s.Subjects {
		if len(sub.Frames) > maxLen {
			maxLen = len(sub.Frames)
		}
		// INV-2: t 강하게 증가
		for i := 1; i < len(sub.Frames); i++ {
			if !(sub.Frames[i].T > sub.Frames[i-1].T) {
				add("invariant", fmt.Sprintf("INV-2 [%s] t가 증가하지 않음: %v -> %v",
					sub.SubjectID, sub.Frames[i-1].T, sub.Frames[i].T))
				break
			}
		}
		for _, f := range sub.Frames {
			ids := map[int]bool{}
			dup := false
			for _, k := range f.Keypoints {
				if ids[k.ID] {
					dup = true
				}
				ids[k.ID] = true
			}
			// INV-3: keypoint id 중복 없음
			if dup {
				add("invariant", fmt.Sprintf("INV-3 [%s t=%v] keypoint id 중복", sub.SubjectID, f.T))
			}
			// INV-4: 토폴로지 완전성(0..expectedN-1 정확히 일치)
			if hasExpected {
				if len(ids) != expectedN || !coversRange(ids, expectedN) {
					add("invariant", fmt.Sprintf("INV-4 [%s t=%v] 토폴로지 불완전(기대 %d개)",
						sub.SubjectID, f.T, expectedN))
				}
			}
			// INV-7: 3d 인데 z 누락이면 오류
			if cp.Dimensions == "3d" {
				for _, k := range f.Keypoints {
					if k.Z == nil {
						add("invariant", fmt.Sprintf("INV-7 [%s t=%v kp=%d] 3d인데 z 누락",
							sub.SubjectID, f.T, k.ID))
					}
				}
			}
		}
	}

	// INV-8: frame_count(존재 시) == 최대 frames 길이
	if cp.FrameCount != nil && *cp.FrameCount != maxLen {
		add("invariant", fmt.Sprintf("INV-8 capture.frame_count=%d != 최대 frames 길이=%d",
			*cp.FrameCount, maxLen))
	}

	return errs
}

func coversRange(ids map[int]bool, n int) bool {
	for i := 0; i < n; i++ {
		if !ids[i] {
			return false
		}
	}
	return true
}
