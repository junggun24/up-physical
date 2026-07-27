package analysis

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

// Engine — 상주(warm) 파이썬 엔진 서버(engine/engine_server.py)와 파이프로 통신하는 클라이언트.
//
// 프로세스를 한 번만 띄워두고 잡마다 요청 1줄을 보내 응답 1줄을 받는다(콜드스타트 제거).
// 레퍼런스는 ref_key 로 엔진이 캐시하므로, 같은 키는 두 번째부터 ref 바이트를 보내지 않는다.
// 호출은 직렬화(mutex). 파이프가 깨지면 프로세스를 재시작하고 1회 재시도한다.
type Engine struct {
	python string
	script string

	mu       sync.Mutex
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	out      *bufio.Reader
	sentRefs map[string]bool // 현재 프로세스에 이미 보낸 ref_key
}

func NewEngine(python, script string) *Engine {
	return &Engine{python: python, script: script, sentRefs: map[string]bool{}}
}

type engineReq struct {
	User   json.RawMessage `json:"user"`
	Ref    json.RawMessage `json:"ref"` // 캐시된 키면 null
	RefKey string          `json:"ref_key"`
}

type engineResp struct {
	OK         bool            `json:"ok"`
	Results    []SubjectResult `json:"results"`
	Comparison json.RawMessage `json:"comparison"`
	Error      string          `json:"error"`
}

// Analyze — userJSON 을 refKey 의 레퍼런스와 비교. refJSON 은 캐시에 없을 때만 전송된다.
func (e *Engine) Analyze(ctx context.Context, userJSON, refJSON []byte, refKey string) (*Result, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	res, err := e.try(userJSON, refJSON, refKey)
	if err != nil {
		// 파이프 손상 가능 → 재시작 후 1회 재시도(이번엔 ref 강제 전송)
		e.stopLocked()
		if serr := e.startLocked(); serr != nil {
			return nil, serr
		}
		res, err = e.try(userJSON, refJSON, refKey)
	}
	return res, err
}

func (e *Engine) try(userJSON, refJSON []byte, refKey string) (*Result, error) {
	if e.cmd == nil {
		if err := e.startLocked(); err != nil {
			return nil, err
		}
	}
	req := engineReq{User: userJSON, RefKey: refKey}
	if !e.sentRefs[refKey] {
		req.Ref = refJSON // 첫 전송
	} // 아니면 nil → JSON null → 엔진이 캐시 사용

	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	if _, err := e.stdin.Write(b); err != nil {
		return nil, err
	}
	line, err := e.out.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("엔진 응답 읽기 실패: %w", err)
	}
	var resp engineResp
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("엔진 응답 파싱 실패: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("엔진 오류: %s", resp.Error)
	}
	if req.Ref != nil {
		e.sentRefs[refKey] = true
	}
	return &Result{Results: resp.Results, Comparison: resp.Comparison}, nil
}

func (e *Engine) startLocked() error {
	cmd := exec.Command(e.python, e.script)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("엔진 기동 실패: %w", err)
	}
	e.cmd = cmd
	e.stdin = stdin
	e.out = bufio.NewReaderSize(stdout, 1<<20)
	e.sentRefs = map[string]bool{} // 새 프로세스 → 캐시 비움
	return nil
}

func (e *Engine) stopLocked() {
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
		_ = e.cmd.Wait()
	}
	e.cmd = nil
	e.stdin = nil
	e.out = nil
}

// Stop — 종료 시 엔진 프로세스 정리.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopLocked()
}
