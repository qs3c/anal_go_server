package oss

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/qs3c/anal_go_server/config"
)

const (
	maxRetries    = 3
	baseRetryWait = 2 * time.Second
)

type Client struct {
	client     *oss.Client
	bucket     *oss.Bucket
	bucketName string
	cdnDomain  string
}

func NewClient(cfg *config.OSSConfig) (*Client, error) {
	client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %w", err)
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &Client{
		client:     client,
		bucket:     bucket,
		bucketName: cfg.BucketName,
		cdnDomain:  cfg.CDNDomain,
	}, nil
}

// UploadDiagram 上传框图 JSON 文件
func (c *Client) UploadDiagram(analysisID int64, data []byte) (string, error) {
	objectKey := fmt.Sprintf("diagrams/%d/%d.json", analysisID, time.Now().Unix())

	err := c.bucket.PutObject(objectKey, bytes.NewReader(data), oss.ContentType("application/json"))
	if err != nil {
		return "", fmt.Errorf("failed to upload diagram: %w", err)
	}

	return c.GetURL(objectKey), nil
}

// UploadDiagramWithRetry 带重试的上传，最多重试 3 次，指数退避
func (c *Client) UploadDiagramWithRetry(analysisID int64, data []byte) (string, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		url, err := c.UploadDiagram(analysisID, data)
		if err == nil {
			return url, nil
		}
		lastErr = err
		if attempt < maxRetries {
			wait := baseRetryWait * (1 << attempt) // 2s, 4s, 8s
			log.Printf("OSS upload attempt %d failed: %v, retrying in %v", attempt+1, err, wait)
			time.Sleep(wait)
		}
	}
	return "", fmt.Errorf("OSS upload failed after %d retries: %w", maxRetries+1, lastErr)
}

// UploadAvatar 上传用户头像
func (c *Client) UploadAvatar(userID int64, data []byte, ext string) (string, error) {
	objectKey := fmt.Sprintf("avatars/%d/%d%s", userID, time.Now().Unix(), ext)

	contentType := getContentType(ext)
	err := c.bucket.PutObject(objectKey, bytes.NewReader(data), oss.ContentType(contentType))
	if err != nil {
		return "", fmt.Errorf("failed to upload avatar: %w", err)
	}

	return c.GetURL(objectKey), nil
}

// UploadFile 上传通用文件
func (c *Client) UploadFile(objectKey string, data []byte, contentType string) (string, error) {
	err := c.bucket.PutObject(objectKey, bytes.NewReader(data), oss.ContentType(contentType))
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return c.GetURL(objectKey), nil
}

// Download 下载文件内容
func (c *Client) Download(objectKey string) ([]byte, error) {
	body, err := c.bucket.GetObject(objectKey)
	if err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}
	return data, nil
}

// Delete 删除文件
func (c *Client) Delete(objectKey string) error {
	err := c.bucket.DeleteObject(objectKey)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}
	return nil
}

// GetURL 获取文件访问 URL
func (c *Client) GetURL(objectKey string) string {
	if c.cdnDomain != "" {
		return fmt.Sprintf("https://%s/%s", c.cdnDomain, objectKey)
	}
	return fmt.Sprintf("https://%s.%s/%s", c.bucketName, c.client.Config.Endpoint, objectKey)
}

// GetSignedURL 生成带签名的临时访问URL（默认1小时有效）
func (c *Client) GetSignedURL(objectKey string, expireSeconds ...int64) (string, error) {
	expire := int64(3600) // 默认1小时
	if len(expireSeconds) > 0 && expireSeconds[0] > 0 {
		expire = expireSeconds[0]
	}

	signedURL, err := c.bucket.SignURL(objectKey, oss.HTTPGet, expire)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	// SDK 默认生成 http:// URL，替换为 https://
	signedURL = strings.Replace(signedURL, "http://", "https://", 1)

	return signedURL, nil
}

// getContentType 根据扩展名获取 Content-Type
func getContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// ExtractObjectKey 从 URL 中提取 object key
func (c *Client) ExtractObjectKey(url string) string {
	// 处理 CDN 域名
	if c.cdnDomain != "" {
		prefix := fmt.Sprintf("https://%s/", c.cdnDomain)
		if strings.HasPrefix(url, prefix) {
			return url[len(prefix):]
		}
	}

	// 处理标准 OSS URL: https://bucket-name.endpoint/path/to/object
	// 或: https://endpoint/bucket-name/path/to/object
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		// 从第4个部分开始是 object key
		return strings.Join(parts[3:], "/")
	}

	return path.Base(url)
}
