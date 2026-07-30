package integration

import (
	"testing"

	"github.com/upphysical/backend/internal/normalize"
)

// 활성 레퍼런스가 바뀌면 점수도 그 레퍼런스를 따라야 한다.
//
// 회귀 대상(2026-07-30 수정): 워커가 동작의 **모든** 레퍼런스 중 최고점을 채택하던 결함.
// 그 상태에서는 활성 레퍼런스가 나빠도 예전 좋은 레퍼런스의 점수가 유지돼,
// 레퍼런스를 늘릴수록 모든 사용자의 점수가 올라가고 "고칠 단 하나"의 근거가 사라졌다.
func TestScoreFollowsActiveReference(t *testing.T) {
	e := setup(t)
	action := uniqueAction("scoring")
	t.Cleanup(func() { e.cleanupAction(t, action) })

	user := e.fixture(t, "valid-forehand-2d.json")
	goodRef := e.fixture(t, "reference-forehand-2d.json")
	// 좌우를 뒤집은 레퍼런스는 같은 사용자 스윙에 대해 훨씬 나쁜 매칭이 된다.
	badRef := normalize.MirrorStream(goodRef)

	e.seedReference(t, action, 1, goodRef, "right")
	goodScore := e.analyze(t, action, user, "")

	// v2 를 등록하면 활성이 v2 로 바뀌고 v1 은 남는다(비활성).
	e.seedReference(t, action, 2, badRef, "right")
	badScore := e.analyze(t, action, user, "")

	t.Logf("활성 v1(좋은 매칭) %.1f → 활성 v2(나쁜 매칭) %.1f", goodScore, badScore)

	if badScore >= goodScore {
		t.Fatalf("활성 레퍼런스가 나빠졌는데 점수가 떨어지지 않았다 "+
			"(비활성 레퍼런스가 채점에 섞이고 있다): good=%.1f bad=%.1f", goodScore, badScore)
	}
	// 결함이 있었다면 badScore 는 goodScore 와 같았다. 유의미한 차이를 요구한다.
	if goodScore-badScore < 5 {
		t.Fatalf("점수 차이가 너무 작다 — 회귀를 감지하지 못할 수 있다: good=%.1f bad=%.1f",
			goodScore, badScore)
	}
}

// 손잡이가 레퍼런스와 다르면 워커가 좌우 반전으로 정규화해야 한다.
//
// 정규화가 없으면 점수가 "자세 차이"가 아니라 "손잡이 차이"를 반영한다.
func TestHandednessNormalization(t *testing.T) {
	e := setup(t)
	action := uniqueAction("handedness")
	t.Cleanup(func() { e.cleanupAction(t, action) })

	ref := e.fixture(t, "reference-forehand-2d.json")
	e.seedReference(t, action, 1, ref, "right")

	righty := e.fixture(t, "valid-forehand-2d.json")
	lefty := e.fixture(t, "valid-forehand-2d-lefty.json")

	baseline := e.analyze(t, action, righty, "right")
	notNormalized := e.analyze(t, action, lefty, "")     // 손잡이 미지정 → 정규화 생략
	normalized := e.analyze(t, action, lefty, "left")    // 반전 적용

	t.Logf("우완 기준선 %.1f · 좌완 미지정 %.1f · 좌완 정규화 %.1f",
		baseline, notNormalized, normalized)

	if notNormalized >= baseline {
		t.Fatalf("정규화 없이도 점수가 떨어지지 않았다 — fixture 가 실제로 반전본이 맞는가: "+
			"baseline=%.1f notNormalized=%.1f", baseline, notNormalized)
	}
	if normalized <= notNormalized {
		t.Fatalf("정규화가 점수를 개선하지 못했다: notNormalized=%.1f normalized=%.1f",
			notNormalized, normalized)
	}
	// 완전히 대칭인 fixture 이므로 정규화 후에는 기준선과 사실상 같아야 한다.
	if diff := baseline - normalized; diff > 0.5 || diff < -0.5 {
		t.Fatalf("정규화 후 점수가 기준선과 다르다 (손잡이 페널티가 남아 있다): "+
			"baseline=%.1f normalized=%.1f", baseline, normalized)
	}
}

// 같은 Idempotency-Key 재업로드는 새 잡을 만들지 않는다.
func TestIdempotentUpload(t *testing.T) {
	e := setup(t)
	action := uniqueAction("idempotency")
	t.Cleanup(func() { e.cleanupAction(t, action) })

	e.seedReference(t, action, 1, e.fixture(t, "reference-forehand-2d.json"), "right")

	user := e.fixture(t, "valid-forehand-2d.json")
	first, second := e.analyzeTwiceSameKey(t, action, user)
	if first != second {
		t.Fatalf("같은 멱등키가 다른 잡을 만들었다: %s vs %s", first, second)
	}
}
