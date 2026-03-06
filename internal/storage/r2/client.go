package r2

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ryank90/lora-trainer-example/internal/config"
	"github.com/ryank90/lora-trainer-example/internal/storage"
)

type Client struct {
	s3Client       *s3.Client
	presignClient  *s3.PresignClient
	bucket         string
	presignDuration time.Duration
}

func NewClient(ctx context.Context, cfg config.StorageConfig) (*Client, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cfg.Endpoint,
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "",
		)),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	return &Client{
		s3Client:       s3Client,
		presignClient:  s3.NewPresignClient(s3Client),
		bucket:         cfg.Bucket,
		presignDuration: cfg.PresignDuration,
	}, nil
}

func (c *Client) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("uploading to R2: %w", err)
	}
	return nil
}

func (c *Client) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := c.s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("downloading from R2: %w", err)
	}
	return output.Body, nil
}

func (c *Client) GeneratePresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry == 0 {
		expiry = c.presignDuration
	}
	req, err := c.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("generating presigned URL: %w", err)
	}
	return req.URL, nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from R2: %w", err)
	}
	return nil
}

func (c *Client) GetMetadata(ctx context.Context, key string) (*storage.ObjectMetadata, error) {
	output, err := c.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("getting metadata from R2: %w", err)
	}

	meta := &storage.ObjectMetadata{
		Key:  key,
		Size: *output.ContentLength,
	}
	if output.ContentType != nil {
		meta.ContentType = *output.ContentType
	}
	if output.LastModified != nil {
		meta.LastModified = *output.LastModified
	}
	return meta, nil
}
