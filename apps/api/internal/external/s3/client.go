package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client defines the interface for S3 operations.
type Client interface {
	UploadAttachment(ctx context.Context, key string, content []byte, contentType string) error
	Download(ctx context.Context, key string) ([]byte, error)
}

// s3Client implements the Client interface using AWS SDK v2.
type s3Client struct {
	client *s3.Client
	bucket string
}

// NewClient creates a new S3 client.
func NewClient(ctx context.Context, region, accessKeyID, secretAccessKey, bucket string) (Client, error) {
	// For production, we usually rely on IAM roles or env vars handled by config.LoadDefaultConfig.
	// However, if keys are explicitly provided, we can use them.
	// Here we stick to standard env var loading if keys are empty, or custom creds if provided.

	var cfg aws.Config
	var err error

	if accessKeyID != "" && secretAccessKey != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     accessKeyID,
					SecretAccessKey: secretAccessKey,
				}, nil
			})),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(region))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &s3Client{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

// UploadAttachment uploads a file to S3.
func (c *s3Client) UploadAttachment(ctx context.Context, key string, content []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload attachment: %w", err)
	}

	return nil
}

// Download retrieves a file from S3.
func (c *s3Client) Download(ctx context.Context, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download attachment: %w", err)
	}
	defer output.Body.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment body: %w", err)
	}

	return buf.Bytes(), nil
}
