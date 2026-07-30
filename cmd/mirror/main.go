// 골격 스트림 좌우 반전 도구.
//
// 용도: (1) 좌완 fixture 생성 — 손잡이 정규화의 회귀 검증에 쓴다,
//       (2) 좌완 코치 영상을 우완 레퍼런스로 등록할 때 사전 변환.
// 변환 구현은 internal/normalize 하나뿐이다(파이프라인·워커·이 도구가 같은 코드를 쓴다).
//
// 사용: go run ./cmd/mirror -in stream.json -out mirrored.json [-session <uuid>]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/upphysical/backend/internal/contract"
	"github.com/upphysical/backend/internal/normalize"
)

func main() {
	in := flag.String("in", "", "입력 골격 스트림 JSON (필수)")
	out := flag.String("out", "", "출력 경로 (필수)")
	session := flag.String("session", "", "출력 스트림의 session_id 를 교체 (선택)")
	flag.Parse()

	if *in == "" || *out == "" {
		log.Fatal("-in, -out 필수")
	}

	raw, err := os.ReadFile(*in)
	if err != nil {
		log.Fatalf("읽기 실패: %v", err)
	}
	st, err := contract.Parse(raw)
	if err != nil {
		log.Fatalf("파싱 실패: %v", err)
	}

	m := normalize.MirrorStream(st)
	if *session != "" {
		m.SessionID = *session
	}
	// 반전 결과도 계약을 만족해야 한다 (경계에서 한 번 더 확인).
	if verrs := contract.Validate(m); len(verrs) > 0 {
		log.Fatalf("반전 결과가 계약 위반: %v", verrs)
	}

	b, err := json.Marshal(m)
	if err != nil {
		log.Fatalf("직렬화 실패: %v", err)
	}
	if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		log.Fatalf("쓰기 실패: %v", err)
	}
	fmt.Printf("OK 좌우 반전: %s → %s (session_id=%s)\n", *in, *out, m.SessionID)
}
