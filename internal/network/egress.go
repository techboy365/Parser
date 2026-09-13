package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/techboy365/Parser/internal/config"
)

type Egress struct {
	cfg    config.NetworkConfig
	client *http.Client
	lastIP string
}

func NewEgress(cfg config.NetworkConfig) *Egress {
	return &Egress{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (e *Egress) Mode() string {
	if e.cfg.Mode == "" {
		return "hotspot"
	}
	return e.cfg.Mode
}

func (e *Egress) Verify(ctx context.Context) (string, error) {
	if !e.cfg.VerifyIPBeforeSearch {
		return "", nil
	}
	url := e.cfg.IPCheckURL
	if url == "" {
		url = "https://api.ipify.org?format=text"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("egress ip check failed: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if ip == "" {
		return "", fmt.Errorf("empty egress ip response")
	}
	e.lastIP = ip
	return ip, nil
}

func (e *Egress) LastIP() string {
	return e.lastIP
}
