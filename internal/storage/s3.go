package storage

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"io"
	"net/http"
	"net/url"
	"time"
)

type S3Config struct {
	Endpoint     string `json:"endpoint"`
	Region       string `json:"region"`
	Bucket       string `json:"bucket"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	SessionToken string `json:"sessionToken,omitempty"`
}
type S3 struct {
	Client *s3.Client
	Bucket string
}

func NewS3(c S3Config) (S3, error) {
	if c.Region == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" {
		return S3{}, errors.New("incomplete S3 settings")
	}
	if c.Endpoint != "" {
		u, e := url.Parse(c.Endpoint)
		if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return S3{}, errors.New("S3 endpoint requires HTTPS")
		}
	}
	options := s3.Options{Region: c.Region, Credentials: credentials.NewStaticCredentialsProvider(c.AccessKey, c.SecretKey, c.SessionToken), HTTPClient: &http.Client{Timeout: 15 * time.Second}, RetryMaxAttempts: 2}
	if c.Endpoint != "" {
		options.BaseEndpoint = aws.String(c.Endpoint)
		options.UsePathStyle = true
	}
	return S3{Client: s3.New(options), Bucket: c.Bucket}, nil
}
func (s S3) Put(ctx context.Context, key string, data []byte) error {
	if !keyPattern.MatchString(key) || len(data) > 4194816 {
		return errors.New("invalid object")
	}
	_, e := s.Client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.Bucket, Key: &key, Body: bytes.NewReader(data), ContentLength: aws.Int64(int64(len(data))), IfNoneMatch: aws.String("*")})
	return e
}
func (s S3) Get(ctx context.Context, key string) ([]byte, error) {
	if !keyPattern.MatchString(key) {
		return nil, errors.New("invalid object")
	}
	out, e := s.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.Bucket, Key: &key})
	if e != nil {
		var api smithy.APIError
		if errors.As(e, &api) && (api.ErrorCode() == "NoSuchKey" || api.ErrorCode() == "NoSuchBucket") {
			return nil, ErrMissing
		}
		return nil, e
	}
	defer out.Body.Close()
	b, e := io.ReadAll(io.LimitReader(out.Body, 4194817))
	if len(b) > 4194816 {
		return nil, errors.New("object too large")
	}
	return b, e
}
func (s S3) Delete(ctx context.Context, key string) error {
	if !keyPattern.MatchString(key) {
		return errors.New("invalid object")
	}
	_, e := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.Bucket, Key: &key})
	return e
}
func (s S3) Check(ctx context.Context) error {
	_, e := s.Client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.Bucket})
	return e
}
