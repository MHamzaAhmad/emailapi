package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
	_, err = io.Copy(buf, output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read attachment body: %w", err)
	}

	return buf.Bytes(), nil
}

// GetObjectTags retrieves tags for an S3 object.
// Used to check GuardDuty malware scan status via the GuardDutyMalwareScanStatus tag.
func (c *s3Client) GetObjectTags(ctx context.Context, key string) (map[string]string, error) {
	input := &s3.GetObjectTaggingInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.GetObjectTagging(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object tags: %w", err)
	}

	tags := make(map[string]string)
	for _, tag := range output.TagSet {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return tags, nil
}

// HeadObject gets object metadata (size, content-type).
func (c *s3Client) HeadObject(ctx context.Context, key string) (*ObjectMeta, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to head object: %w", err)
	}

	meta := &ObjectMeta{}
	if output.ContentLength != nil {
		meta.SizeBytes = *output.ContentLength
	}
	if output.ContentType != nil {
		meta.ContentType = *output.ContentType
	}

	return meta, nil
}

// DeleteObject removes an object from S3.
func (c *s3Client) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
