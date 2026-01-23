package oss

import (
	"bytes"
	"fmt"
	"path"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"github.com/qs3c/anal_go_server/config"
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
	if c.cdnDomain != "" {
		prefix := fmt.Sprintf("https://%s/", c.cdnDomain)
		if len(url) > len(prefix) {
			return url[len(prefix):]
		}
	}
	return path.Base(url)
}
