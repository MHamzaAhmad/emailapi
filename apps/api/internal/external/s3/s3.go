package s3

import "context"

// ObjectMeta contains metadata about an S3 object.
type ObjectMeta struct {
	SizeBytes   int64
	ContentType string
}

// Client defines the interface for S3 operations.
// This interface is meant to be easily mockable for testing.
type Client interface {
	// UploadAttachment uploads a file to S3.
	UploadAttachment(ctx context.Context, key string, content []byte, contentType string) error

	// Download retrieves a file from S3.
	Download(ctx context.Context, key string) ([]byte, error)

	// GetObjectTags retrieves tags for an S3 object (for GuardDuty scan status).
	GetObjectTags(ctx context.Context, key string) (map[string]string, error)

	// HeadObject gets object metadata (size, content-type).
	HeadObject(ctx context.Context, key string) (*ObjectMeta, error)

	// DeleteObject removes an object from S3.
	DeleteObject(ctx context.Context, key string) error
}
