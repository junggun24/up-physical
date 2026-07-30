// Package normalize — 분석 입력의 좌표 정규화 (순수 기하 변환).
//
// 경계: 채점(DTW·점수 산출)은 엔진의 일이고 여기서는 좌표만 다룬다. 엔진 경계 정책을
// 어기지 않는다 — 점수 계산 로직을 Go 로 복제하는 게 아니다.
// 또한 **저장된 원본 스트림은 절대 바꾸지 않는다**: 변환은 엔진 호출 직전에만 적용한다
// (원본 보존은 재분석·회귀의 전제다).
package normalize

import (
	"github.com/upphysical/backend/internal/contract"
)

// mirrorPairs — blazepose_33 의 좌/우 대응 키포인트.
//
// 좌우 반전은 x 뒤집기만으로 끝나지 않는다. 뒤집으면 해부학적 좌/우가 서로의 자리로 가므로
// **쌍을 이루는 id 를 함께 교환**해야 "타격 손"이 사용자와 레퍼런스에서 같은 id 가 된다.
// (교환을 빼먹으면 좌완 사용자의 비타격 손을 레퍼런스의 타격 손과 비교하게 된다.)
var mirrorPairs = [][2]int{
	{1, 4}, {2, 5}, {3, 6}, // 눈: inner · 중앙 · outer
	{7, 8},                 // 귀
	{9, 10},                // 입꼬리
	{11, 12},               // 어깨
	{13, 14},               // 팔꿈치
	{15, 16},               // 손목
	{17, 18},               // 새끼손가락
	{19, 20},               // 검지
	{21, 22},               // 엄지
	{23, 24},               // 골반
	{25, 26},               // 무릎
	{27, 28},               // 발목
	{29, 30},               // 발꿈치
	{31, 32},               // 발끝
}
// 0(코)은 대응쌍이 없다 — 자기 자신으로 매핑된다.

// partnerOf — id → 반전 시 좌표를 가져올 상대 id.
var partnerOf = func() map[int]int {
	m := make(map[int]int, 33)
	for _, p := range mirrorPairs {
		m[p[0]] = p[1]
		m[p[1]] = p[0]
	}
	return m
}()

// Partner — 좌/우 대응 id (없으면 자기 자신). 테스트·문서화 목적으로 공개한다.
func Partner(id int) int {
	if p, ok := partnerOf[id]; ok {
		return p
	}
	return id
}

// MirrorStream — 좌우를 반전한 새 스트림을 반환한다(입력은 변경하지 않는다).
//
// 용도: 좌완 사용자를 우완 레퍼런스와 비교할 때(또는 반대) 손잡이 차이를 제거한다.
// 레퍼런스를 손잡이별로 2배 모으는 대신 변환 한 번으로 해결한다.
//
// x 뒤집기는 좌표계에 따라 다르다:
//   - image_normalized: x ∈ [0,1] → 1-x
//   - world_metric / root_relative: 원점 기준 → -x
func MirrorStream(s *contract.Stream) *contract.Stream {
	flip := flipperFor(s.Capture.CoordinateSpace)

	out := *s // 얕은 복사 후 subjects 를 새로 만든다
	out.Subjects = make([]contract.Subject, len(s.Subjects))

	for si, sub := range s.Subjects {
		newSub := sub
		newSub.Frames = make([]contract.Frame, len(sub.Frames))

		for fi, f := range sub.Frames {
			// 키포인트가 id 순서라는 보장이 없으므로 id 로 색인한다.
			byID := make(map[int]contract.Keypoint, len(f.Keypoints))
			for _, kp := range f.Keypoints {
				byID[kp.ID] = kp
			}

			newKps := make([]contract.Keypoint, len(f.Keypoints))
			for ki, kp := range f.Keypoints {
				src, ok := byID[Partner(kp.ID)]
				if !ok {
					src = kp // 대응 id 가 없는 불완전 프레임 — 자기 좌표를 뒤집는다
				}
				newKps[ki] = contract.Keypoint{
					ID:         kp.ID, // 위치의 id 는 유지하고 좌표만 상대에게서 가져온다
					X:          flip(src.X),
					Y:          src.Y, // 좌우 반전은 y 를 바꾸지 않는다
					Z:          src.Z,
					Visibility: src.Visibility,
				}
			}
			newSub.Frames[fi] = contract.Frame{T: f.T, Keypoints: newKps}
		}
		out.Subjects[si] = newSub
	}
	return &out
}

func flipperFor(space string) func(float64) float64 {
	if space == "image_normalized" {
		return func(x float64) float64 { return 1 - x }
	}
	return func(x float64) float64 { return -x }
}
