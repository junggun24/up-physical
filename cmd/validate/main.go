// validate — 골격 스트림 JSON 파일을 계약(스키마 필수값 + INV-1..8)으로 검증하는 CLI.
//
// 하네스 검증 루프(.harness)의 도구: fixture 가 실제 경계 검증 코드(internal/contract)를
// 통과하는지 확인한다. 문서가 아니라 코드가 진실이 되도록, 검증 로직을 복제하지 않고
// API 와 동일한 contract.Parse/Validate 를 그대로 사용한다.
//
// 사용법:
//
//	go run ./cmd/validate <stream.json> [...]
//	go run ./cmd/validate -expect-invalid <bad-stream.json> [...]
//
// 종료코드: 0 = 기대와 일치, 1 = 불일치(유효해야 하는데 위반 / 무효여야 하는데 통과).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/upphysical/backend/internal/contract"
)

func main() {
	expectInvalid := flag.Bool("expect-invalid", false, "파일이 검증에 실패해야 통과(무효 fixture 용)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "사용법: validate [-expect-invalid] <stream.json> [...]")
		os.Exit(2)
	}

	fail := false
	for _, path := range flag.Args() {
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL  %s: 읽기 실패: %v\n", path, err)
			fail = true
			continue
		}

		var verrs []contract.ValErr
		st, err := contract.Parse(raw)
		if err != nil {
			verrs = []contract.ValErr{{Path: "$", Message: err.Error()}}
		} else {
			verrs = contract.Validate(st)
		}

		valid := len(verrs) == 0
		switch {
		case valid && !*expectInvalid:
			fmt.Printf("OK    %s (계약 통과)\n", path)
		case !valid && *expectInvalid:
			fmt.Printf("OK    %s (기대대로 무효: %d건, 첫 오류: %s)\n", path, len(verrs), verrs[0].Error())
		case valid && *expectInvalid:
			fmt.Printf("FAIL  %s: 무효를 기대했으나 검증 통과\n", path)
			fail = true
		default:
			fmt.Printf("FAIL  %s: 계약 위반 %d건\n", path, len(verrs))
			for _, e := range verrs {
				fmt.Printf("      - %s\n", e.Error())
			}
			fail = true
		}
	}
	if fail {
		os.Exit(1)
	}
}
