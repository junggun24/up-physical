package normalize

import (
	"math"
	"testing"

	"github.com/upphysical/backend/internal/contract"
)

func kp(id int, x, y float64) contract.Keypoint {
	return contract.Keypoint{ID: id, X: x, Y: y, Visibility: 0.9}
}

// stream33 — 계약을 만족하는 최소 스트림(33개 키포인트, 2프레임).
func stream33(space string) *contract.Stream {
	frames := make([]contract.Frame, 0, 2)
	for i := 0; i < 2; i++ {
		kps := make([]contract.Keypoint, 0, 33)
		for id := 0; id < 33; id++ {
			// id 마다 다른 x 를 주어 교환이 일어났는지 식별 가능하게 한다
			kps = append(kps, kp(id, 0.01*float64(id), 0.5))
		}
		frames = append(frames, contract.Frame{T: float64(i) / 30.0, Keypoints: kps})
	}
	n := 2
	return &contract.Stream{
		SchemaVersion: "1.0",
		SessionID:     "55555555-5555-4555-8555-555555555555",
		Capture: contract.Capture{
			Source: "on_device", Model: "blazepose", ModelVariant: "full", ModelVersion: "1",
			FPS: 30, CoordinateSpace: space, Dimensions: "2d",
			KeypointTopology: "blazepose_33", FrameCount: &n,
		},
		Subjects: []contract.Subject{{SubjectID: "p1", Frames: frames}},
	}
}

func findKP(f contract.Frame, id int) contract.Keypoint {
	for _, k := range f.Keypoints {
		if k.ID == id {
			return k
		}
	}
	panic("id 없음")
}

func TestPartnerPairs(t *testing.T) {
	// 손목: 좌(15) ↔ 우(16) — 타격 손 정렬의 핵심
	if Partner(15) != 16 || Partner(16) != 15 {
		t.Fatal("손목 좌우 대응이 틀렸다")
	}
	if Partner(0) != 0 {
		t.Fatal("코(0)는 대응쌍이 없어 자기 자신이어야 한다")
	}
	// 모든 id 가 왕복해야 한다
	for id := 0; id < 33; id++ {
		if Partner(Partner(id)) != id {
			t.Fatalf("id %d 왕복 실패", id)
		}
	}
}

func TestMirrorSwapsLeftRightAndFlipsX(t *testing.T) {
	s := stream33("image_normalized")
	orig := s.Subjects[0].Frames[0]
	m := MirrorStream(s)
	got := m.Subjects[0].Frames[0]

	// 좌측 손목(15) 자리에는 원래 우측 손목(16)의 좌표가 뒤집혀 들어와야 한다
	wantX := 1 - findKP(orig, 16).X
	if math.Abs(findKP(got, 15).X-wantX) > 1e-9 {
		t.Fatalf("id15 의 x 가 id16 에서 오지 않았다: got %v want %v", findKP(got, 15).X, wantX)
	}
	// 반대 방향도 성립
	if math.Abs(findKP(got, 16).X-(1-findKP(orig, 15).X)) > 1e-9 {
		t.Fatal("id16 의 x 가 id15 에서 오지 않았다")
	}
	// y 는 변하지 않는다
	if findKP(got, 15).Y != findKP(orig, 16).Y {
		t.Fatal("좌우 반전이 y 를 바꿨다")
	}
	// 코는 자기 x 만 뒤집힌다
	if math.Abs(findKP(got, 0).X-(1-findKP(orig, 0).X)) > 1e-9 {
		t.Fatal("코의 x 뒤집기가 틀렸다")
	}
}

func TestMirrorIsInvolution(t *testing.T) {
	s := stream33("image_normalized")
	back := MirrorStream(MirrorStream(s))
	for _, id := range []int{0, 11, 15, 16, 32} {
		a := findKP(s.Subjects[0].Frames[0], id)
		b := findKP(back.Subjects[0].Frames[0], id)
		if math.Abs(a.X-b.X) > 1e-9 || a.Y != b.Y {
			t.Fatalf("두 번 반전하면 원본이어야 한다 (id %d): %v vs %v", id, a, b)
		}
	}
}

func TestMirrorDoesNotMutateInput(t *testing.T) {
	s := stream33("image_normalized")
	before := findKP(s.Subjects[0].Frames[0], 15).X
	_ = MirrorStream(s)
	if findKP(s.Subjects[0].Frames[0], 15).X != before {
		t.Fatal("입력 스트림이 변경됐다 — 원본 보존 위반")
	}
}

func TestMirroredStreamStillPassesContract(t *testing.T) {
	s := stream33("image_normalized")
	m := MirrorStream(s)
	if errs := contract.Validate(m); len(errs) > 0 {
		t.Fatalf("반전 결과가 계약을 위반했다: %v", errs)
	}
}

func TestMirrorWorldMetricUsesNegation(t *testing.T) {
	s := stream33("world_metric")
	orig := findKP(s.Subjects[0].Frames[0], 16).X
	m := MirrorStream(s)
	if math.Abs(findKP(m.Subjects[0].Frames[0], 15).X-(-orig)) > 1e-9 {
		t.Fatal("world_metric 은 원점 기준 -x 여야 한다")
	}
}

func TestMirrorHandlesUnorderedKeypoints(t *testing.T) {
	s := stream33("image_normalized")
	f := &s.Subjects[0].Frames[0]
	f.Keypoints[15], f.Keypoints[16] = f.Keypoints[16], f.Keypoints[15] // 순서 뒤섞기
	m := MirrorStream(s)
	// 순서와 무관하게 id 기준으로 교환되어야 한다
	if math.Abs(findKP(m.Subjects[0].Frames[0], 15).X-(1-0.16)) > 1e-9 {
		t.Fatalf("키포인트 순서가 뒤섞여도 id 기준으로 교환해야 한다: %v",
			findKP(m.Subjects[0].Frames[0], 15).X)
	}
}
