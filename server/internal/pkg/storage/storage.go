// Package storage 封装 S3 兼容的对象存储（开发环境 RustFS，生产环境 OSS / COS），见技术选型。
//
// 公有桶存放商品封面、店铺头像等公开图片，匿名可读；私有桶存放虚拟商品文件，只能通过签名 URL 下载。
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	"mkclou/server/internal/pkg/config"
)

type Client struct {
	s3      *s3.Client
	cfg     config.S3Config
	baseURL string
}

func New(cfg config.S3Config) *Client {
	c := s3.New(s3.Options{
		Region:       cfg.Region,
		BaseEndpoint: aws.String(cfg.Endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		UsePathStyle: cfg.UsePathStyle,
		// RustFS 等 S3 兼容服务不一定支持新版 SDK 默认的请求校验和
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	base := cfg.PublicBaseURL
	if base == "" {
		base = fmt.Sprintf("%s/%s", strings.TrimRight(cfg.Endpoint, "/"), cfg.PublicBucket)
	}
	return &Client{s3: c, cfg: cfg, baseURL: strings.TrimRight(base, "/")}
}

// PublicURL 返回公有桶中对象的访问地址。
func (c *Client) PublicURL(key string) string { return c.baseURL + "/" + key }

// PutPublic 上传公开对象（如图片）。
func (c *Client) PutPublic(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.cfg.PublicBucket),
		Key:           aws.String(key),
		Body:          body,
		ContentLength: aws.Int64(size),
		ContentType:   aws.String(contentType),
		CacheControl:  aws.String("public, max-age=31536000, immutable"), // key 含随机串，内容不会变
	})
	return err
}

// EnsureBuckets 创建公有桶与私有桶（已存在时跳过），并为公有桶设置匿名只读策略。可重复执行。
func (c *Client) EnsureBuckets(ctx context.Context) error {
	for _, b := range []string{c.cfg.PublicBucket, c.cfg.PrivateBucket} {
		if err := c.ensureBucket(ctx, b); err != nil {
			return err
		}
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, c.cfg.PublicBucket)
	if _, err := c.s3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(c.cfg.PublicBucket),
		Policy: aws.String(policy),
	}); err != nil {
		return fmt.Errorf("set public bucket policy: %w", err)
	}
	return nil
}

func (c *Client) ensureBucket(ctx context.Context, name string) error {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(name)})
	if err == nil {
		return nil
	}
	if _, err := c.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(name)}); err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "BucketAlreadyOwnedByYou" || apiErr.ErrorCode() == "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("create bucket %s: %w", name, err)
	}
	return nil
}
