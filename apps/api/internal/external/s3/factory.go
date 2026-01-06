package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// BucketName represents a logical bucket name.
type BucketName string

// Predefined bucket names for easier management.
const (
	BucketAttachments BucketName = "attachments"
	BucketInbound     BucketName = "inbound"
)

// Factory provides bucket-scoped S3 clients.
// It holds the AWS configuration and a registry of bucket mappings.
type Factory struct {
	client  *s3.Client
	buckets map[BucketName]string // logical name -> actual bucket name
	mu      sync.RWMutex
}

// Ensure Factory implements FactoryInterface
var _ FactoryInterface = (*Factory)(nil)

// NewFactory creates a new S3 factory with AWS credentials.
func NewFactory(ctx context.Context, region, accessKeyID, secretAccessKey string) (*Factory, error) {
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

	return &Factory{
		client:  s3.NewFromConfig(cfg),
		buckets: make(map[BucketName]string),
	}, nil
}

// RegisterBucket registers a logical bucket name to an actual S3 bucket.
func (f *Factory) RegisterBucket(name BucketName, actualBucket string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buckets[name] = actualBucket
}

// Bucket returns a bucket-scoped client for the given logical name.
func (f *Factory) Bucket(name BucketName) Client {
	f.mu.RLock()
	defer f.mu.RUnlock()

	actualBucket, ok := f.buckets[name]
	if !ok {
		// Return a client that will error on all operations
		return &errorClient{err: fmt.Errorf("bucket %q not registered", name)}
	}

	return &bucketClient{
		client: f.client,
		bucket: actualBucket,
	}
}

// bucketClient implements Client for a specific bucket.
type bucketClient struct {
	client *s3.Client
	bucket string
}

// UploadAttachment uploads a file to S3.
func (c *bucketClient) UploadAttachment(ctx context.Context, key string, content []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	}

	_, err := c.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to %s/%s: %w", c.bucket, key, err)
	}

	return nil
}

// Download retrieves a file from S3.
func (c *bucketClient) Download(ctx context.Context, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	output, err := c.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download from %s/%s: %w", c.bucket, key, err)
	}
	defer output.Body.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, output.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return buf.Bytes(), nil
}

// GetObjectTags retrieves tags for an S3 object.
func (c *bucketClient) GetObjectTags(ctx context.Context, key string) (map[string]string, error) {
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

// HeadObject gets object metadata.
func (c *bucketClient) HeadObject(ctx context.Context, key string) (*ObjectMeta, error) {
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
func (c *bucketClient) DeleteObject(ctx context.Context, key string) error {
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

// errorClient is returned when a bucket is not registered.
type errorClient struct {
	err error
}

func (e *errorClient) UploadAttachment(ctx context.Context, key string, content []byte, contentType string) error {
	return e.err
}

func (e *errorClient) Download(ctx context.Context, key string) ([]byte, error) {
	return nil, e.err
}

func (e *errorClient) GetObjectTags(ctx context.Context, key string) (map[string]string, error) {
	return nil, e.err
}

func (e *errorClient) HeadObject(ctx context.Context, key string) (*ObjectMeta, error) {
	return nil, e.err
}

func (e *errorClient) DeleteObject(ctx context.Context, key string) error {
	return e.err
}
