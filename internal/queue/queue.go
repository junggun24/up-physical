// Package queue — Postgres 기반 잡 큐(별도 브로커 없이).
//
// at-least-once + 멱등 처리 원칙. SELECT … FOR UPDATE SKIP LOCKED 로 워커 다중 인스턴스가
// 같은 잡을 중복 선점하지 않도록 원자적으로 클레임한다.
// 참조구현 worker.claim_next 의 운영 버전.
package queue

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Claimed struct {
	JobID       string
	SessionID   string
	ReferenceID string
}

// ClaimNext — queued 잡 하나를 processing 으로 원자적 선점. 없으면 (nil,nil).
func ClaimNext(ctx context.Context, pool *pgxpool.Pool) (*Claimed, error) {
	var c Claimed
	var refID *string
	err := pool.QueryRow(ctx, `
		UPDATE analysis_jobs SET status='processing', attempts=attempts+1, started_at=now()
		WHERE id = (
			SELECT id FROM analysis_jobs
			WHERE status='queued'
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id::text, session_id::text, reference_id::text`).
		Scan(&c.JobID, &c.SessionID, &refID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if refID != nil {
		c.ReferenceID = *refID
	}
	return &c, nil
}
