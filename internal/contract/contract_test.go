package contract

import (
	"strings"
	"testing"
)

// makeValid — coco_17 · 2d · 1 subject · 3프레임의 최소 유효 스트림.
func makeValid() *Stream {
	frames := make([]Frame, 3)
	for i := range frames {
		kps := make([]Keypoint, 17)
		for k := range kps {
			kps[k] = Keypoint{ID: k, X: 0.5, Y: 0.5, Visibility: 0.9}
		}
		frames[i] = Frame{T: float64(i) / 30.0, Keypoints: kps}
	}
	fc := 3
	return &Stream{
		SchemaVersion: "1.0",
		SessionID:     "22222222-2222-4222-8222-222222222222",
		Capture: Capture{
			Source:           "on_device",
			Model:            "movenet",
			FPS:              30,
			CoordinateSpace:  "image_normalized",
			Dimensions:       "2d",
			KeypointTopology: "coco_17",
			FrameCount:       &fc,
		},
		Subjects: []Subject{{SubjectID: "player-1", Frames: frames}},
	}
}

// hasErr — 오류 목록에 부분 문자열 포함 여부.
func hasErr(errs []ValErr, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

func TestValidate_ValidStreamPasses(t *testing.T) {
	if errs := Validate(makeValid()); len(errs) != 0 {
		t.Fatalf("유효 스트림이 실패: %v", errs)
	}
}

func TestValidate_RequiredFields(t *testing.T) {
	s := makeValid()
	s.SchemaVersion = ""
	s.SessionID = ""
	errs := Validate(s)
	if !hasErr(errs, "schema_version") || !hasErr(errs, "session_id") {
		t.Fatalf("필수값 누락 미검출: %v", errs)
	}
}

func TestValidate_EnumAndFPS(t *testing.T) {
	s := makeValid()
	s.Capture.Source = "webcam"
	s.Capture.CoordinateSpace = "pixel"
	s.Capture.Dimensions = "4d"
	s.Capture.FPS = 0
	errs := Validate(s)
	for _, want := range []string{"capture.source", "capture.coordinate_space", "capture.dimensions", "capture.fps"} {
		if !hasErr(errs, want) {
			t.Errorf("%s 위반 미검출: %v", want, errs)
		}
	}
}

func TestValidate_INV1_DuplicateSubjectID(t *testing.T) {
	s := makeValid()
	s.Subjects = append(s.Subjects, s.Subjects[0]) // 같은 subject_id 재사용
	if errs := Validate(s); !hasErr(errs, "INV-1") {
		t.Fatalf("INV-1 미검출: %v", errs)
	}
}

func TestValidate_INV2_NonIncreasingT(t *testing.T) {
	s := makeValid()
	s.Subjects[0].Frames[2].T = s.Subjects[0].Frames[0].T
	if errs := Validate(s); !hasErr(errs, "INV-2") {
		t.Fatalf("INV-2 미검출: %v", errs)
	}
}

func TestValidate_INV3_DuplicateKeypointID(t *testing.T) {
	s := makeValid()
	// id 16 → 0 으로 바꿔 중복 생성(동시에 토폴로지도 불완전해짐)
	s.Subjects[0].Frames[0].Keypoints[16].ID = 0
	errs := Validate(s)
	if !hasErr(errs, "INV-3") {
		t.Fatalf("INV-3 미검출: %v", errs)
	}
	if !hasErr(errs, "INV-4") {
		t.Fatalf("INV-4 (동반 위반) 미검출: %v", errs)
	}
}

func TestValidate_INV4_IncompleteTopology(t *testing.T) {
	s := makeValid()
	f := &s.Subjects[0].Frames[0]
	f.Keypoints = f.Keypoints[:16] // 키포인트 1개 누락
	if errs := Validate(s); !hasErr(errs, "INV-4") {
		t.Fatalf("INV-4 미검출: %v", errs)
	}
}

func TestValidate_INV4_BlazeposeDefaultTopology(t *testing.T) {
	// model=blazepose 이고 topology 미지정이면 blazepose_33 을 기대해야 한다.
	s := makeValid()
	s.Capture.Model = "blazepose"
	s.Capture.KeypointTopology = "" // 미지정 → 33개 기대, coco 17개라 위반
	if errs := Validate(s); !hasErr(errs, "INV-4") {
		t.Fatalf("blazepose 기본 토폴로지(33) 미적용: %v", errs)
	}
}

func TestValidate_INV7_Missing3DZ(t *testing.T) {
	s := makeValid()
	s.Capture.Dimensions = "3d" // z 는 전부 nil 인 상태
	if errs := Validate(s); !hasErr(errs, "INV-7") {
		t.Fatalf("INV-7 미검출: %v", errs)
	}
}

func TestValidate_INV8_FrameCountMismatch(t *testing.T) {
	s := makeValid()
	wrong := 99
	s.Capture.FrameCount = &wrong
	if errs := Validate(s); !hasErr(errs, "INV-8") {
		t.Fatalf("INV-8 미검출: %v", errs)
	}
}

func TestParse_BadJSON(t *testing.T) {
	if _, err := Parse([]byte("{not json")); err == nil {
		t.Fatal("잘못된 JSON 을 통과시킴")
	}
}
