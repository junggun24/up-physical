// Package auth — 기본(자체) 인증: bcrypt 비밀번호 해시 + HS256 JWT 발급/검증.
//
// 표준 라이브러리만으로 JWT(HS256)를 직접 구현(외부 JWT 의존성 없음).
// 운영 단계에서 관리형 IdP(Firebase 등)로 이전하면 이 패키지는 토큰 검증만 교체하면 된다.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ── 비밀번호 ──

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// ── JWT (HS256) ──

type Claims struct {
	Sub   string `json:"sub"`   // users.id
	Email string `json:"email"`
	Exp   int64  `json:"exp"`   // 만료(Unix 초)
}

func Sign(sub, email, secret string, ttl time.Duration) (string, error) {
	header := b64(`{"alg":"HS256","typ":"JWT"}`)
	cb, err := json.Marshal(Claims{Sub: sub, Email: email, Exp: time.Now().Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	seg := header + "." + b64(string(cb))
	return seg + "." + sign(seg, secret), nil
}

func Verify(token, secret string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	seg := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(sign(seg, secret))) {
		return nil, errors.New("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.Exp {
		return nil, errors.New("token expired")
	}
	return &c, nil
}

func b64(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

func sign(seg, secret string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(seg))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
