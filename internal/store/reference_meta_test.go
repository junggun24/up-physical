package store

import "testing"

// 권리 근거 없는 레퍼런스가 등록되지 않는다는 것이 이 스키마의 존재 이유다.
// DB 없이 검증 가능한 경계를 유닛으로 고정한다 (DB CHECK 제약은 2차 방어선).
func TestReferenceMetaValidate(t *testing.T) {
	cases := []struct {
		name    string
		meta    ReferenceMeta
		wantErr bool
	}{
		{
			name: "허락받은 코치 영상 — 통과",
			meta: ReferenceMeta{
				SourceKind:   "permission",
				RightsBasis:  "코치 DM 허락 2026-07-30 (캡처: drive.example/abc)",
				ProviderName: "sgga_tennis",
				Handedness:   "right",
			},
		},
		{
			name: "자체 촬영 — 통과",
			meta: ReferenceMeta{SourceKind: "self_recorded", RightsBasis: "2026-07-30 자체 촬영, 코트 A"},
		},
		{
			name:    "권리 근거 없음 — 거부",
			meta:    ReferenceMeta{SourceKind: "permission", RightsBasis: ""},
			wantErr: true,
		},
		{
			name:    "권리 근거가 공백뿐 — 거부",
			meta:    ReferenceMeta{SourceKind: "permission", RightsBasis: "   \t "},
			wantErr: true,
		},
		{
			name:    "source_kind 누락 — 거부",
			meta:    ReferenceMeta{RightsBasis: "근거 있음"},
			wantErr: true,
		},
		{
			name:    "source_kind 오타 — 거부",
			meta:    ReferenceMeta{SourceKind: "youtube", RightsBasis: "근거 있음"},
			wantErr: true,
		},
		{
			name:    "handedness 오값 — 거부",
			meta:    ReferenceMeta{SourceKind: "synthetic", RightsBasis: "합성", Handedness: "both"},
			wantErr: true,
		},
		{
			name: "handedness 미지정 — 통과(선택 필드)",
			meta: ReferenceMeta{SourceKind: "synthetic", RightsBasis: "합성"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.meta.Validate()
			if tc.wantErr && err == nil {
				t.Fatalf("거부되어야 하는데 통과했다: %+v", tc.meta)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("통과해야 하는데 거부됐다: %v", err)
			}
		})
	}
}

func TestNullIfEmpty(t *testing.T) {
	if nullIfEmpty("") != nil || nullIfEmpty("  ") != nil {
		t.Fatal("빈 값은 NULL 이어야 한다")
	}
	if v := nullIfEmpty("side"); v == nil || *v != "side" {
		t.Fatal("값이 있으면 그대로 보존해야 한다")
	}
}
