package kameleo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type Fingerprint struct {
	ID string `json:"id"`
}

type Profile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CreateProfileRequest struct {
	FingerprintID string `json:"fingerprintId"`
	Name          string `json:"name"`
	Language      string `json:"language,omitempty"`
	Storage       string `json:"storage,omitempty"`
}

func NewClient(endpoint string) *Client {
	return &Client{
		baseURL: strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) VerifyEngineReady(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/general/healthcheck", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("kameleo engine not reachable at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("kameleo healthcheck failed: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) SearchFingerprints(ctx context.Context, deviceType, browserProduct string) ([]Fingerprint, error) {
	url := fmt.Sprintf("%s/fingerprints?deviceType=%s&browserProduct=%s", c.baseURL, deviceType, browserProduct)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search fingerprints: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var fps []Fingerprint
	if err := json.NewDecoder(resp.Body).Decode(&fps); err != nil {
		return nil, fmt.Errorf("decode fingerprints: %w", err)
	}
	if len(fps) == 0 {
		return nil, fmt.Errorf("no fingerprints found for device=%s browser=%s", deviceType, browserProduct)
	}
	return fps, nil
}

func (c *Client) CreateProfile(ctx context.Context, req CreateProfileRequest) (*Profile, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/profiles/new", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create profile: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var profile Profile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &profile, nil
}

func (c *Client) StartProfile(ctx context.Context, profileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/profiles/%s/start", c.baseURL, profileID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("start profile: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) StopProfile(ctx context.Context, profileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/profiles/%s/stop", c.baseURL, profileID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop profile: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) DeleteProfile(ctx context.Context, profileID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/profiles/%s", c.baseURL, profileID), nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete profile: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) PlaywrightCDPEndpoint(profileID string) string {
	host := strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://")
	return fmt.Sprintf("ws://%s/playwright/%s", host, profileID)
}
