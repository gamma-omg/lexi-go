package image

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

type RemoteStore struct {
	Url       string
	FieldName string
	FileName  string
	client    *http.Client
}

func NewRemoteStore(url, fieldName, fileName string) *RemoteStore {
	return &RemoteStore{
		Url:       url,
		FieldName: fieldName,
		FileName:  fileName,
		client:    &http.Client{},
	}
}

type saveImageResponse struct {
	ImageURL string `json:"image_url"`
}

func (s *RemoteStore) SaveImage(ctx context.Context, img io.Reader) (*url.URL, error) {
	var body strings.Builder
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile(s.FieldName, s.FileName)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	_, err = io.Copy(part, img)
	if err != nil {
		return nil, fmt.Errorf("copy image data: %w", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Url, strings.NewReader(body.String()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var saveResp saveImageResponse
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&saveResp)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	imgUrl, err := url.Parse(saveResp.ImageURL)
	if err != nil {
		return nil, fmt.Errorf("parse image URL: %w", err)
	}

	return imgUrl, nil
}
