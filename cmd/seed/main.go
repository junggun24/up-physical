// 레퍼런스 시드 도구.
//
// "정답 예시" 골격 스트림(코치 스윙)을 오브젝트 스토리지에 올리고 DB에 활성 레퍼런스로 등록한다.
// 이게 있어야 /v1/sessions 가 (sport, action)의 활성 레퍼런스와 비교해 분석할 수 있다.
//
// 사용:
//   set -a; source ../deploy/.env; set +a
//   go run ./cmd/seed -sport tennis -action forehand -version 1 -file ../smoke/reference_forehand.json
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/upphysical/backend/internal/contract"
	"github.com/upphysical/backend/internal/objstore"
	"github.com/upphysical/backend/internal/store"
)

func main() {
	sport := flag.String("sport", "tennis", "종목")
	action := flag.String("action", "forehand", "동작")
	version := flag.Int("version", 1, "레퍼런스 버전")
	file := flag.String("file", "", "레퍼런스 골격 스트림 JSON 경로 (필수)")

	// 출처·권리 (권리 근거는 필수 — 근거 없는 레퍼런스는 등록되지 않는다)
	sourceKind := flag.String("source-kind", "", "취득 경로 (필수): self_recorded|permission|cc_licensed|synthetic")
	rights := flag.String("rights", "", "권리 근거 (필수): 허락 증빙 링크·라이선스 URL·촬영 메모")
	provider := flag.String("provider", "", "제공자(코치·채널명) — 앱 크레딧 표기용")
	attribution := flag.String("attribution", "", "표기 의무 문구 (CC-BY 등)")
	handedness := flag.String("handedness", "", "right|left")
	level := flag.String("level", "", "coach|advanced|intermediate 등")
	angle := flag.String("angle", "", "side|front|diagonal")
	notes := flag.String("notes", "", "비고")
	flag.Parse()

	if *file == "" {
		log.Fatal("-file 필수")
	}
	meta := store.ReferenceMeta{
		SourceKind:   *sourceKind,
		RightsBasis:  *rights,
		ProviderName: *provider,
		Attribution:  *attribution,
		Handedness:   *handedness,
		SkillLevel:   *level,
		CameraAngle:  *angle,
		Notes:        *notes,
	}
	// 스토리지에 올리기 전에 거부한다 (부작용 없이 실패).
	if err := meta.Validate(); err != nil {
		log.Fatalf("레퍼런스 메타데이터 오류: %v", err)
	}

	ctx := context.Background()

	raw, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("파일 읽기 실패: %v", err)
	}
	st, err := contract.Parse(raw)
	if err != nil {
		log.Fatalf("파싱 실패: %v", err)
	}
	if verrs := contract.Validate(st); len(verrs) > 0 {
		log.Fatalf("레퍼런스 검증 실패: %v", verrs)
	}

	dataStore, err := store.New(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer dataStore.Close()

	host, ssl := parseEndpoint(mustEnv("S3_ENDPOINT"))
	obj, err := objstore.New(objstore.Config{
		Endpoint:  host,
		AccessKey: mustEnv("S3_ACCESS_KEY"),
		SecretKey: mustEnv("S3_SECRET_KEY"),
		UseSSL:    ssl,
	})
	if err != nil {
		log.Fatalf("objstore: %v", err)
	}

	key := fmt.Sprintf("%s/%s/v%d/reference.json", *sport, *action, *version)
	if _, err := obj.PutBytes(ctx, store.BucketReferences, key, raw); err != nil {
		log.Fatalf("스토리지 업로드 실패: %v", err)
	}

	refID, err := dataStore.RegisterReference(ctx, *sport, *action, *version, st, store.BucketReferences, key, meta)
	if err != nil {
		log.Fatalf("등록 실패: %v", err)
	}

	fmt.Printf("OK 레퍼런스 등록됨: %s/%s v%d (id=%s, key=%s)\n", *sport, *action, *version, refID, key)
	fmt.Printf("   출처=%s · 권리근거=%q", meta.SourceKind, meta.RightsBasis)
	if meta.ProviderName != "" {
		fmt.Printf(" · 제공자=%s", meta.ProviderName)
	}
	fmt.Println()
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("환경변수 %s 필요", k)
	}
	return v
}

func parseEndpoint(raw string) (host string, ssl bool) {
	ssl = strings.HasPrefix(raw, "https://")
	host = strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	host = strings.TrimSuffix(host, "/")
	return host, ssl
}
