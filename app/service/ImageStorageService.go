package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/leanote/leanote/app/lea"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type ImageStorageService struct {
}

type imageStorageConfig struct {
	Provider      string
	Bucket        string
	Endpoint      string
	Region        string
	AccessKey     string
	SecretKey     string
	PublicBaseUrl string
	ObjectPrefix  string
}

func (this *ImageStorageService) IsRemoteEnabled() bool {
	provider := this.config().Provider
	return provider != "" && provider != "local"
}

func (this *ImageStorageService) PutNoteImage(userId, filename string, data []byte) (publicUrl, objectKey string, size int64, err error) {
	cfg := this.config()
	if cfg.Provider == "" || cfg.Provider == "local" {
		err = errors.New("image storage provider is local")
		return
	}

	objectKey = this.buildObjectKey(cfg.ObjectPrefix, userId, filename)
	size = int64(len(data))

	switch cfg.Provider {
	case "huawei_obs", "huawei", "obs":
		publicUrl, err = this.putHuaweiOBS(cfg, objectKey, data)
	case "aliyun_oss", "tencent_cos", "aws_s3":
		err = fmt.Errorf("image storage provider %s is not implemented", cfg.Provider)
	default:
		err = fmt.Errorf("unknown image storage provider %s", cfg.Provider)
	}
	return
}

func (this *ImageStorageService) config() imageStorageConfig {
	return imageStorageConfig{
		Provider:      strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageProvider")),
		Bucket:        strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageBucket")),
		Endpoint:      normalizeStorageEndpoint(configService.GetGlobalStringConfig("imageStorageEndpoint")),
		Region:        strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageRegion")),
		AccessKey:     strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageAccessKey")),
		SecretKey:     strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageSecretKey")),
		PublicBaseUrl: strings.TrimRight(strings.TrimSpace(configService.GetGlobalStringConfig("imageStoragePublicBaseUrl")), "/"),
		ObjectPrefix:  strings.Trim(strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageObjectPrefix")), "/"),
	}
}

func (this *ImageStorageService) buildObjectKey(prefix, userId, filename string) string {
	now := time.Now()
	parts := []string{prefix, "note-images", lea.Digest3(userId), userId, now.Format("2006"), now.Format("01"), now.Format("02"), filename}
	cleanParts := []string{}
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	return path.Join(cleanParts...)
}

func (this *ImageStorageService) putHuaweiOBS(cfg imageStorageConfig, objectKey string, data []byte) (string, error) {
	if cfg.Bucket == "" || cfg.Endpoint == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return "", errors.New("huawei obs config requires bucket, endpoint, accessKey and secretKey")
	}

	contentType := http.DetectContentType(data)
	if contentType == "application/octet-stream" {
		contentType = contentTypeByExt(objectKey)
	}

	objectUrl := "https://" + cfg.Bucket + "." + cfg.Endpoint + "/" + escapeObjectKey(objectKey)
	req, err := http.NewRequest("PUT", objectUrl, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-obs-acl", "public-read")

	stringToSign := strings.Join([]string{
		"PUT",
		"",
		contentType,
		date,
		"x-obs-acl:public-read",
		"/" + cfg.Bucket + "/" + objectKey,
	}, "\n")
	req.Header.Set("Authorization", "OBS "+cfg.AccessKey+":"+hmacSha1Base64(cfg.SecretKey, stringToSign))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("huawei obs upload failed: %s %s", resp.Status, strings.TrimSpace(string(body)))
	}

	if cfg.PublicBaseUrl != "" {
		return cfg.PublicBaseUrl + "/" + escapeObjectKey(objectKey), nil
	}
	return objectUrl, nil
}

func normalizeStorageEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return strings.Trim(endpoint, "/")
}

func escapeObjectKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func hmacSha1Base64(secret, data string) string {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func contentTypeByExt(name string) string {
	lowerName := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lowerName, ".jpg"), strings.HasSuffix(lowerName, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lowerName, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerName, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lowerName, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lowerName, ".webp"):
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
