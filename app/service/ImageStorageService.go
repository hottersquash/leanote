package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/leanote/leanote/app/lea"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"sort"
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

// IsRemoteObjectPath reports whether a stored File.Path is a cloud object key
// (as opposed to a local files/ or public/ path).
func (this *ImageStorageService) IsRemoteObjectPath(path string) bool {
	path = strings.TrimLeft(strings.TrimSpace(path), "/")
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "files/") || strings.HasPrefix(path, "public/") {
		return false
	}
	// Absolute local paths are not remote object keys.
	if strings.HasPrefix(path, "/") || strings.Contains(path, ":\\") {
		return false
	}
	return this.IsRemoteEnabled()
}

// PublicURL builds the browser-facing URL for a cloud object key.
func (this *ImageStorageService) PublicURL(objectKey string) string {
	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if objectKey == "" {
		return ""
	}
	cfg := this.config()
	objectUrl := this.objectURL(cfg, objectKey)
	return buildPublicUrl(cfg, objectUrl, objectKey)
}

func (this *ImageStorageService) objectURL(cfg imageStorageConfig, objectKey string) string {
	escapedKey := escapeObjectKey(objectKey)
	switch cfg.Provider {
	case "huawei_obs", "huawei", "obs":
		if cfg.Bucket == "" || cfg.Endpoint == "" {
			return ""
		}
		return "https://" + cfg.Bucket + "." + cfg.Endpoint + "/" + escapedKey
	case "aliyun_oss", "aliyun":
		endpoint, err := resolveAliyunEndpoint(cfg)
		if err != nil || cfg.Bucket == "" {
			return ""
		}
		return "https://" + cfg.Bucket + "." + endpoint + "/" + escapedKey
	case "tencent_cos", "tencent":
		host, err := resolveTencentHost(cfg)
		if err != nil {
			return ""
		}
		return "https://" + host + "/" + escapedKey
	case "aws_s3", "aws", "s3":
		host, _, err := resolveAwsHostAndRegion(cfg)
		if err != nil {
			return ""
		}
		return "https://" + host + "/" + escapedKey
	default:
		if cfg.PublicBaseUrl != "" {
			return cfg.PublicBaseUrl + "/" + escapedKey
		}
		return ""
	}
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
	case "aliyun_oss", "aliyun":
		publicUrl, err = this.putAliyunOSS(cfg, objectKey, data)
	case "tencent_cos", "tencent":
		publicUrl, err = this.putTencentCOS(cfg, objectKey, data)
	case "aws_s3", "aws", "s3":
		publicUrl, err = this.putAwsS3(cfg, objectKey, data)
	default:
		err = fmt.Errorf("unknown image storage provider %s", cfg.Provider)
	}
	return
}

func (this *ImageStorageService) config() imageStorageConfig {
	bucket := strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageBucket"))
	return imageStorageConfig{
		Provider:      strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageProvider")),
		Bucket:        bucket,
		Endpoint:      normalizeStorageEndpoint(configService.GetGlobalStringConfig("imageStorageEndpoint"), bucket),
		Region:        strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageRegion")),
		AccessKey:     strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageAccessKey")),
		SecretKey:     strings.TrimSpace(configService.GetGlobalStringConfig("imageStorageSecretKey")),
		PublicBaseUrl: strings.TrimRight(strings.TrimSpace(configService.GetGlobalStringConfig("imageStoragePublicBaseUrl")), "/"),
		ObjectPrefix:  normalizeObjectPrefix(configService.GetGlobalStringConfig("imageStorageObjectPrefix")),
	}
}

func (this *ImageStorageService) buildObjectKey(prefix, userId, filename string) string {
	now := time.Now()
	parts := []string{prefix, "note-images", lea.Digest3(userId), userId, now.Format("2006"), now.Format("01"), now.Format("02"), filename}
	cleanParts := []string{}
	for _, part := range parts {
		part = strings.Trim(part, "/")
		part = strings.TrimSpace(part)
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
	if strings.HasPrefix(cfg.Endpoint, cfg.Bucket+".") {
		return "", fmt.Errorf("huawei obs endpoint looks like a bucket domain (%s); use only the regional endpoint such as obs.cn-north-4.myhuaweicloud.com", cfg.Endpoint)
	}

	contentType := resolveContentType(data, objectKey)
	host := cfg.Bucket + "." + cfg.Endpoint
	escapedKey := escapeObjectKey(objectKey)
	objectUrl := "https://" + host + "/" + escapedKey
	req, err := http.NewRequest("PUT", objectUrl, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	// Keep the already-escaped path; Go would otherwise decode/re-encode and break OBS signing.
	req.URL.Opaque = "//" + host + "/" + escapedKey
	req.Host = host

	// Prefer x-obs-date for signing (Date slot in StringToSign is empty). Still send Date
	// for compatibility with proxies that expect it.
	obsDate := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", obsDate)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("x-obs-acl", "public-read")
	req.Header.Set("x-obs-date", obsDate)

	// CanonicalizedResource must use the URI-encoded object name (same as request path).
	stringToSign := strings.Join([]string{
		"PUT",
		"",
		contentType,
		"",
		"x-obs-acl:public-read",
		"x-obs-date:" + obsDate,
		"/" + cfg.Bucket + "/" + escapedKey,
	}, "\n")
	req.Header.Set("Authorization", "OBS "+cfg.AccessKey+":"+hmacSha1Base64(cfg.SecretKey, stringToSign))

	return doObjectUpload(req, "huawei obs", objectUrl, cfg, objectKey)
}

func (this *ImageStorageService) putAliyunOSS(cfg imageStorageConfig, objectKey string, data []byte) (string, error) {
	endpoint, err := resolveAliyunEndpoint(cfg)
	if err != nil {
		return "", err
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return "", errors.New("aliyun oss config requires bucket, endpoint or region, accessKey and secretKey")
	}

	contentType := resolveContentType(data, objectKey)
	host := cfg.Bucket + "." + endpoint
	escapedKey := escapeObjectKey(objectKey)
	objectUrl := "https://" + host + "/" + escapedKey
	req, err := http.NewRequest("PUT", objectUrl, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.URL.Opaque = "//" + host + "/" + escapedKey
	req.Host = host

	date := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))
	req.Header.Set("x-oss-acl", "public-read")

	stringToSign := strings.Join([]string{
		"PUT",
		"",
		contentType,
		date,
		"x-oss-acl:public-read",
		"/" + cfg.Bucket + "/" + escapedKey,
	}, "\n")
	req.Header.Set("Authorization", "OSS "+cfg.AccessKey+":"+hmacSha1Base64(cfg.SecretKey, stringToSign))

	return doObjectUpload(req, "aliyun oss", objectUrl, cfg, objectKey)
}

func (this *ImageStorageService) putTencentCOS(cfg imageStorageConfig, objectKey string, data []byte) (string, error) {
	host, err := resolveTencentHost(cfg)
	if err != nil {
		return "", err
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return "", errors.New("tencent cos config requires bucket, region or endpoint, accessKey and secretKey")
	}

	contentType := resolveContentType(data, objectKey)
	objectUrl := "https://" + host + "/" + escapeObjectKey(objectKey)
	req, err := http.NewRequest("PUT", objectUrl, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-cos-acl", "public-read")

	now := time.Now().Unix()
	signTime := fmt.Sprintf("%d;%d", now-60, now+3600)
	httpURI := "/" + escapeObjectKey(objectKey)

	headerValues := map[string]string{
		"content-type": contentType,
		"host":         host,
		"x-cos-acl":    "public-read",
	}
	headerList, headerString := buildCosHeaderSignParts(headerValues)

	httpString := strings.Join([]string{
		"put",
		httpURI,
		"",
		headerString,
		"",
	}, "\n")
	stringToSign := strings.Join([]string{
		"sha1",
		signTime,
		sha1Hex(httpString),
		"",
	}, "\n")
	signKey := hmacSha1Bytes(cfg.SecretKey, signTime)
	signature := hex.EncodeToString(hmacSha1Bytes(string(signKey), stringToSign))

	req.Header.Set("Authorization", fmt.Sprintf(
		"q-sign-algorithm=sha1&q-ak=%s&q-sign-time=%s&q-key-time=%s&q-header-list=%s&q-url-param-list=&q-signature=%s",
		cfg.AccessKey,
		signTime,
		signTime,
		headerList,
		signature,
	))

	return doObjectUpload(req, "tencent cos", objectUrl, cfg, objectKey)
}

func (this *ImageStorageService) putAwsS3(cfg imageStorageConfig, objectKey string, data []byte) (string, error) {
	host, region, err := resolveAwsHostAndRegion(cfg)
	if err != nil {
		return "", err
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return "", errors.New("aws s3 config requires bucket, region or endpoint, accessKey and secretKey")
	}

	contentType := resolveContentType(data, objectKey)
	objectUrl := "https://" + host + "/" + escapeObjectKey(objectKey)
	req, err := http.NewRequest("PUT", objectUrl, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	dateStamp := now.Format("20060102")
	payloadHash := "UNSIGNED-PAYLOAD"

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-amz-acl", "public-read")
	req.Header.Set("x-amz-content-sha256", payloadHash)
	req.Header.Set("x-amz-date", amzDate)

	canonicalURI := "/" + escapeObjectKey(objectKey)
	canonicalHeaders := strings.Join([]string{
		"content-type:" + contentType,
		"host:" + host,
		"x-amz-acl:public-read",
		"x-amz-content-sha256:" + payloadHash,
		"x-amz-date:" + amzDate,
	}, "\n") + "\n"
	signedHeaders := "content-type;host;x-amz-acl;x-amz-content-sha256;x-amz-date"
	canonicalRequest := strings.Join([]string{
		"PUT",
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		sha256Hex(canonicalRequest),
	}, "\n")

	signingKey := deriveAwsSigningKey(cfg.SecretKey, dateStamp, region)
	signature := hex.EncodeToString(hmacSha256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		cfg.AccessKey,
		credentialScope,
		signedHeaders,
		signature,
	))

	return doObjectUpload(req, "aws s3", objectUrl, cfg, objectKey)
}

var objectUploadClient = &http.Client{
	Timeout: 120 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func doObjectUpload(req *http.Request, providerName, objectUrl string, cfg imageStorageConfig, objectKey string) (string, error) {
	resp, err := objectUploadClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("%s upload failed: %s %s", providerName, resp.Status, formatObjectStorageError(body))
	}

	return buildPublicUrl(cfg, objectUrl, objectKey), nil
}

func formatObjectStorageError(body []byte) string {
	text := strings.TrimSpace(string(body))
	if text == "" {
		return ""
	}
	code := xmlTagValue(text, "Code")
	message := xmlTagValue(text, "Message")
	stringToSign := xmlTagValue(text, "StringToSign")
	parts := make([]string, 0, 3)
	if code != "" {
		parts = append(parts, code)
	}
	if message != "" {
		parts = append(parts, message)
	}
	if stringToSign != "" {
		parts = append(parts, "StringToSign="+strings.ReplaceAll(stringToSign, "\n", "\\n"))
	}
	if len(parts) == 0 {
		return text
	}
	out := strings.Join(parts, " | ")
	if code == "SignatureDoesNotMatch" {
		out += " | Check Access Key / Secret Key pair; Endpoint must be regional host only (e.g. obs.cn-north-4.myhuaweicloud.com); Object Prefix must not contain spaces (use leanote/img not 'leanote / img')."
	}
	return out
}

func xmlTagValue(xmlBody, tag string) string {
	start := strings.Index(xmlBody, "<"+tag+">")
	end := strings.Index(xmlBody, "</"+tag+">")
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	start += len(tag) + 2
	return strings.TrimSpace(xmlBody[start:end])
}

func buildPublicUrl(cfg imageStorageConfig, objectUrl, objectKey string) string {
	if cfg.PublicBaseUrl != "" {
		return cfg.PublicBaseUrl + "/" + escapeObjectKey(objectKey)
	}
	return objectUrl
}

func resolveAliyunEndpoint(cfg imageStorageConfig) (string, error) {
	if cfg.Endpoint != "" {
		return cfg.Endpoint, nil
	}
	if cfg.Region == "" {
		return "", errors.New("aliyun oss config requires endpoint or region")
	}
	return "oss-" + cfg.Region + ".aliyuncs.com", nil
}

func resolveTencentHost(cfg imageStorageConfig) (string, error) {
	if cfg.Endpoint != "" {
		endpoint := cfg.Endpoint
		if strings.HasPrefix(endpoint, cfg.Bucket+".") {
			return endpoint, nil
		}
		if strings.HasPrefix(endpoint, "cos.") {
			return cfg.Bucket + "." + endpoint, nil
		}
		return endpoint, nil
	}
	if cfg.Region == "" {
		return "", errors.New("tencent cos config requires endpoint or region")
	}
	return cfg.Bucket + ".cos." + cfg.Region + ".myqcloud.com", nil
}

func resolveAwsHostAndRegion(cfg imageStorageConfig) (host string, region string, err error) {
	region = cfg.Region
	if cfg.Endpoint != "" {
		host = cfg.Bucket + "." + cfg.Endpoint
		if region == "" {
			region = inferAwsRegionFromEndpoint(cfg.Endpoint)
		}
		if region == "" {
			err = errors.New("aws s3 config requires region when endpoint does not include a region")
		}
		return
	}
	if region == "" {
		err = errors.New("aws s3 config requires endpoint or region")
		return
	}
	if region == "us-east-1" {
		host = cfg.Bucket + ".s3.amazonaws.com"
		return
	}
	host = cfg.Bucket + ".s3." + region + ".amazonaws.com"
	return
}

func inferAwsRegionFromEndpoint(endpoint string) string {
	parts := strings.Split(endpoint, ".")
	for i, part := range parts {
		if part == "s3" && i+1 < len(parts) && parts[i+1] != "amazonaws" {
			return parts[i+1]
		}
	}
	return ""
}

func buildCosHeaderSignParts(headers map[string]string) (headerList string, headerString string) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+url.QueryEscape(headers[key]))
	}
	return strings.Join(keys, ";"), strings.Join(pairs, "&")
}

func deriveAwsSigningKey(secretKey, dateStamp, region string) []byte {
	kDate := hmacSha256([]byte("AWS4"+secretKey), dateStamp)
	kRegion := hmacSha256(kDate, region)
	kService := hmacSha256(kRegion, "s3")
	return hmacSha256(kService, "aws4_request")
}

func normalizeStorageEndpoint(endpoint, bucket string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.Trim(endpoint, "/")
	if endpoint == "" {
		return ""
	}
	// Drop any path the user may have pasted (bucket/object paths break Host parsing).
	if i := strings.Index(endpoint, "/"); i >= 0 {
		endpoint = endpoint[:i]
	}
	// Drop default HTTPS port if present.
	endpoint = strings.TrimSuffix(endpoint, ":443")
	bucket = strings.TrimSpace(bucket)
	if bucket != "" {
		prefix := bucket + "."
		if strings.HasPrefix(endpoint, prefix) {
			endpoint = strings.TrimPrefix(endpoint, prefix)
		}
	}
	return endpoint
}

// NormalizeObjectPrefix turns "leanote / img" into "leanote/img".
func NormalizeObjectPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	parts := strings.Split(prefix, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			clean = append(clean, part)
		}
	}
	return strings.Join(clean, "/")
}

func normalizeObjectPrefix(prefix string) string {
	return NormalizeObjectPrefix(prefix)
}

func escapeObjectKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func resolveContentType(data []byte, objectKey string) string {
	contentType := http.DetectContentType(data)
	if contentType == "application/octet-stream" {
		contentType = contentTypeByExt(objectKey)
	}
	return contentType
}

func hmacSha1Base64(secret, data string) string {
	return base64.StdEncoding.EncodeToString(hmacSha1Bytes(secret, data))
}

func hmacSha1Bytes(secret, data string) []byte {
	mac := hmac.New(sha1.New, []byte(secret))
	mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha1Hex(data string) string {
	sum := sha1.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

func sha256Hex(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func hmacSha256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
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
