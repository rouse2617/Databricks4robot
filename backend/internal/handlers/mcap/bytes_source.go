package mcap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
)

// BytesSource abstracts backing storage for MCAP bytes.
// It is injected so handlers can be unit-tested without real GCS.
type BytesSource interface {
	// Stat returns the total object size in bytes and an optional content type.
	Stat(ctx context.Context, gsPath string) (size int64, contentType string, err error)
	// OpenRange opens a reader for [start, start+length) bytes.
	OpenRange(ctx context.Context, gsPath string, start, length int64) (io.ReadCloser, error)
	// OpenFull opens a reader for the whole object.
	OpenFull(ctx context.Context, gsPath string) (io.ReadCloser, error)
}

type gcsBytesSource struct {
	client *storage.Client
}

func NewGCSBytesSource(client *storage.Client) (BytesSource, error) {
	if client == nil {
		return nil, errors.New("nil storage client")
	}
	return &gcsBytesSource{client: client}, nil
}

func (s *gcsBytesSource) Stat(ctx context.Context, gsPath string) (int64, string, error) {
	bucket, object, ok := parseGSURI(gsPath)
	if !ok {
		return 0, "", fmt.Errorf("invalid gs path: %q", gsPath)
	}
	attrs, err := s.client.Bucket(bucket).Object(object).Attrs(ctx)
	if err != nil {
		return 0, "", err
	}
	return attrs.Size, attrs.ContentType, nil
}

func (s *gcsBytesSource) OpenRange(ctx context.Context, gsPath string, start, length int64) (io.ReadCloser, error) {
	bucket, object, ok := parseGSURI(gsPath)
	if !ok {
		return nil, fmt.Errorf("invalid gs path: %q", gsPath)
	}
	return s.client.Bucket(bucket).Object(object).NewRangeReader(ctx, start, length)
}

func (s *gcsBytesSource) OpenFull(ctx context.Context, gsPath string) (io.ReadCloser, error) {
	bucket, object, ok := parseGSURI(gsPath)
	if !ok {
		return nil, fmt.Errorf("invalid gs path: %q", gsPath)
	}
	return s.client.Bucket(bucket).Object(object).NewReader(ctx)
}

// parseGSURI splits gs://bucket/object/key into bucket and object name.
// Copied from services/mcap-preview/internal/gcsrs for backend-local reuse.
func parseGSURI(uri string) (bucket, object string, ok bool) {
	const prefix = "gs://"
	if !strings.HasPrefix(uri, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(uri, prefix)
	i := strings.IndexByte(rest, '/')
	if i <= 0 || i >= len(rest)-1 {
		return "", "", false
	}
	return rest[:i], rest[i+1:], true
}
