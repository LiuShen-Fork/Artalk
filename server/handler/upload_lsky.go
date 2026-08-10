package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/utils"
)

type lskyUploadResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    struct {
		URL   string            `json:"url"`
		Links map[string]string `json:"links"`
	} `json:"data"`
}

func uploadToLsky(conf config.LskyConf, file io.Reader, filename string) (string, error) {
	endpoint, err := lskyUploadEndpoint(conf.BaseURL)
	if err != nil {
		return "", err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileWriter, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create lsky multipart file: %w", err)
	}
	if _, err := io.Copy(fileWriter, file); err != nil {
		return "", fmt.Errorf("write lsky multipart file: %w", err)
	}

	if permission := strings.TrimSpace(conf.Permission); permission != "" {
		switch strings.ToLower(permission) {
		case "public":
			_ = writer.WriteField("permission", "1")
		case "private":
			_ = writer.WriteField("permission", "0")
		default:
			_ = writer.WriteField("permission", permission)
		}
	}
	if conf.StrategyID > 0 {
		_ = writer.WriteField("strategy_id", strconv.Itoa(conf.StrategyID))
	}
	if conf.AlbumID > 0 {
		_ = writer.WriteField("album_id", strconv.Itoa(conf.AlbumID))
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close lsky multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("create lsky upload request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token := strings.TrimSpace(conf.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request lsky upload API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read lsky upload response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("lsky upload API returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var result lskyUploadResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode lsky upload response: %w", err)
	}
	if !result.Status {
		if result.Message == "" {
			result.Message = "lsky upload failed"
		}
		return "", errors.New(result.Message)
	}

	imgURL := result.Data.Links["url"]
	if imgURL == "" {
		imgURL = result.Data.URL
	}
	if imgURL == "" || !utils.ValidateURL(imgURL) {
		return "", fmt.Errorf("lsky upload response missing valid image URL")
	}
	return imgURL, nil
}

func lskyUploadEndpoint(baseURL string) (string, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "", fmt.Errorf("lsky base URL is required")
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid lsky base URL %q", baseURL)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("lsky base URL must not contain query or fragment")
	}

	if !strings.HasSuffix(parsed.Path, "/api/v1") {
		baseURL += "/api/v1"
	}
	return baseURL + "/upload", nil
}
