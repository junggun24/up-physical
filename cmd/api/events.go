package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// hub — 잡 완료 이벤트 팬아웃.
//
// API 프로세스 하나가 Postgres 채널('job_done')을 LISTEN 하고, 들어온 알림을
// 해당 job_id 를 기다리는 SSE 구독자들에게 전달한다. 워커(별도 프로세스)가 완료 시
// pg_notify 로 알린다(폴링 제거).
type hub struct {
	mu   sync.Mutex
	subs map[string]map[chan string]struct{} // job_id -> 구독 채널 집합(값=status)
}

func newHub() *hub {
	return &hub{subs: map[string]map[chan string]struct{}{}}
}

func (h *hub) sub(jobID string) chan string {
	ch := make(chan string, 1)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[jobID] == nil {
		h.subs[jobID] = map[chan string]struct{}{}
	}
	h.subs[jobID][ch] = struct{}{}
	return ch
}

func (h *hub) unsub(jobID string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.subs[jobID]; m != nil {
		delete(m, ch)
		if len(m) == 0 {
			delete(h.subs, jobID)
		}
	}
}

func (h *hub) publish(jobID, status string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[jobID] {
		select {
		case ch <- status:
		default: // 버퍼 차 있으면 스킵(구독자가 곧 읽음)
		}
	}
}

// listen — 전용 커넥션 하나로 LISTEN job_done. 끊기면 재연결.
func (h *hub) listen(ctx context.Context, pool *pgxpool.Pool) {
	for {
		if err := h.listenOnce(ctx, pool); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("LISTEN 재연결: %v", err)
			time.Sleep(time.Second)
		}
	}
}

func (h *hub) listenOnce(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN job_done"); err != nil {
		return err
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		jobID, status, _ := strings.Cut(n.Payload, ":")
		h.publish(jobID, status)
	}
}

// handleJobEvents — SSE 스트림. 잡이 끝나면 status 이벤트를 보내고 닫는다.
//   GET /v1/jobs/{job_id}/events  → event: status / data: succeeded|failed|timeout
func (s *server) handleJobEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if _, ok := s.resolveUserID(ctx, r); !ok {
		writeProblem(w, http.StatusUnauthorized, "unauthorized", "인증 필요(로그인)")
		return
	}
	jobID := r.PathValue("job_id")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProblem(w, http.StatusInternalServerError, "no_stream", "스트리밍 미지원")
		return
	}

	ch := s.hub.sub(jobID)
	defer s.hub.unsub(jobID, ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	send := func(status string) {
		fmt.Fprintf(w, "event: status\ndata: %s\n\n", status)
		flusher.Flush()
	}

	// 경합 방지: 구독 후 현재 상태를 즉시 확인(이미 끝났을 수 있음)
	if st, err := s.st.JobStatus(ctx, jobID); err == nil && (st == "succeeded" || st == "failed") {
		send(st)
		return
	}

	deadline := time.After(90 * time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case st := <-ch:
			if st == "succeeded" || st == "failed" {
				send(st)
				return
			}
		case <-deadline:
			send("timeout")
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n") // 주석 이벤트(연결 유지)
			flusher.Flush()
		}
	}
}
