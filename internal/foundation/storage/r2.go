package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/divinecoid/one-backend/internal/foundation/config"
)

// Storage abstracts binary object storage (Cloudflare R2, S3-compatible).
type Storage interface {
	Upload(ctx context.Context, key string, content []byte, contentType string) (url string, err error)
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Enabled() bool
}

// NewStorage returns a real R2-backed Storage when cfg is fully configured,
// or a no-op fallback otherwise.
func NewStorage(cfg config.R2Config) Storage {
	if !cfg.Enabled() {
		return &noopStorage{}
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return &noopStorage{}
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint())
		o.UsePathStyle = true
	})

	return &r2Storage{client: client, cfg: cfg}
}

type r2Storage struct {
	client *s3.Client
	cfg    config.R2Config
}

func (s *r2Storage) Enabled() bool {
	return true
}

func (s *r2Storage) Upload(ctx context.Context, key string, content []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload object to R2: %w", err)
	}

	if s.cfg.PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", s.cfg.PublicBaseURL, key), nil
	}
	return key, nil
}

func (s *r2Storage) Download(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download object from R2: %w", err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object body from R2: %w", err)
	}
	return data, nil
}

func (s *r2Storage) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.BucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object from R2: %w", err)
	}
	return nil
}

// noopStorage is used when R2 credentials are not configured. Callers must
// check Enabled() and fall back to metadata-only behavior instead of calling
// these methods.
type noopStorage struct{}

func (s *noopStorage) Enabled() bool {
	return false
}

func (s *noopStorage) Upload(ctx context.Context, key string, content []byte, contentType string) (string, error) {
	return "", fmt.Errorf("object storage is not configured")
}

func (s *noopStorage) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, fmt.Errorf("object storage is not configured")
}

func (s *noopStorage) Delete(ctx context.Context, key string) error {
	return fmt.Errorf("object storage is not configured")
}
