// Package objstore — S3 호환 오브젝트 스토리지(MinIO) 추상화.
//
// 대용량 골격 스트림 원본은 DB가 아니라 여기에 저장하고, DB엔 버킷/키만 남긴다.
// 참조구현(services/objectstore.py)의 put_json/get_json 을 minio-go 로 이식.
// 키 규약(db/STORAGE.md): sessions/{id}/stream.json, {sport}/{action}/v{n}/reference.json
package objstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	client *minio.Client
}

type Config struct {
	Endpoint  string // 예: "localhost:9000"
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// PutMeta — 저장 결과 메타(무결성 확인용).
type PutMeta struct {
	Bucket string
	Key    string
	Bytes  int
	SHA256 string
}

func New(cfg Config) (*Store, error) {
	cli, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client 생성 실패: %w", err)
	}
	return &Store{client: cli}, nil
}

// EnsureBucket — 없으면 생성(멱등).
func (s *Store) EnsureBucket(ctx context.Context, bucket string) error {
	ok, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !ok {
		return s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}
	return nil
}

// PutBytes — 원시 바이트 저장. 스트림은 검증된 그대로(재직렬화 없이) 보존한다.
func (s *Store) PutBytes(ctx context.Context, bucket, key string, data []byte) (PutMeta, error) {
	if err := s.EnsureBucket(ctx, bucket); err != nil {
		return PutMeta{}, err
	}
	_, err := s.client.PutObject(ctx, bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		return PutMeta{}, fmt.Errorf("put 실패 %s/%s: %w", bucket, key, err)
	}
	sum := sha256.Sum256(data)
	return PutMeta{Bucket: bucket, Key: key, Bytes: len(data), SHA256: hex.EncodeToString(sum[:])}, nil
}

// GetBytes — 원시 바이트 로드.
func (s *Store) GetBytes(ctx context.Context, bucket, key string) ([]byte, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get 실패 %s/%s: %w", bucket, key, err)
	}
	defer obj.Close()
	return io.ReadAll(obj)
}
