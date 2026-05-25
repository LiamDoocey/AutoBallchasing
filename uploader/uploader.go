package uploader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	baseURL   = "https://ballchasing.com/api"
	uploadURL = baseURL + "/v2/upload"
	pingURL   = baseURL + "/"
)

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityUnlisted Visibility = "unlisted"
	VisibilityPrivate  Visibility = "private"
)

type Uploader struct {
	apiKey     string
	visibility Visibility
	uploadURL  string
	pingURL    string
	client     *http.Client
}

type UploadResult struct {
	Filename  string
	ReplayID  string
	Location  string
	Success   bool
	Duplicate bool
	Error     string
	Time      time.Time
}

type uploadResponse struct {
	ID       string `json:"id"`
	Location string `json:"location"`
	Error    string `json:"error"`
}

func New(apiKey string, visibility Visibility) *Uploader {
	return &Uploader{
		apiKey:     apiKey,
		visibility: visibility,
		uploadURL:  uploadURL,
		pingURL:    pingURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func NewWithURLs(apiKey string, visibility Visibility, uploadURL, pingURL string) *Uploader {
	return &Uploader{
		apiKey:     apiKey,
		visibility: visibility,
		uploadURL:  uploadURL,
		pingURL:    pingURL,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

func (u *Uploader) Ping() error {
	req, err := http.NewRequest("GET", u.pingURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", u.apiKey)

	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach ballchasing: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid API key")
	default:
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
}

func (u *Uploader) Upload(path string) UploadResult {
	result := UploadResult{
		Filename: filepath.Base(path),
		Time:     time.Now(),
	}

	f, err := os.Open(path)
	if err != nil {
		result.Error = fmt.Sprintf("could not open file: %v", err)
		return result
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	ext := filepath.Ext(path)
	part, err := writer.CreateFormFile("file", timestamp+ext)
	if err != nil {
		result.Error = fmt.Sprintf("could not create form: %v", err)
		return result
	}
	if _, err = io.Copy(part, f); err != nil {
		result.Error = fmt.Sprintf("could not read file: %v", err)
		return result
	}
	writer.Close()

	url := fmt.Sprintf("%s?visibility=%s", u.uploadURL, u.visibility)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		result.Error = fmt.Sprintf("could not create request: %v", err)
		return result
	}
	req.Header.Set("Authorization", u.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := u.client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("upload failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	var apiResp uploadResponse
	json.NewDecoder(resp.Body).Decode(&apiResp)

	switch resp.StatusCode {
	case http.StatusConflict:
		result.Success = true
		result.Duplicate = true
		result.ReplayID = apiResp.ID
		result.Location = apiResp.Location
	case http.StatusCreated:
		result.Success = true
		result.ReplayID = apiResp.ID
		result.Location = apiResp.Location
		// Set the title to ISO timestamp
		timestamp := time.Now().Format("2006-01-02T15:04:05")
		if err := u.patchTitle(apiResp.ID, timestamp); err != nil {
			log.Printf("could not patch title: %v", err)
		}
	default:
		if apiResp.Error != "" {
			result.Error = apiResp.Error
		} else {
			result.Error = fmt.Sprintf("server returned %d", resp.StatusCode)
		}
	}

	return result
}

func (u *Uploader) patchTitle(replayID string, title string) error {
	body := fmt.Sprintf(`{"title":%q}`, title)
	req, err := http.NewRequest("PATCH",
		fmt.Sprintf("%s/replays/%s", baseURL, replayID),
		bytes.NewBufferString(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", u.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
