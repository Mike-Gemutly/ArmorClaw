package subprocess

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CDPHealthChecker struct {
	port       string
	httpClient *http.Client
}

type CDPVersionResponse struct {
	Browser         string `json:"Browser"`
	ProtocolVersion string `json:"Protocol-Version"`
	UserAgent       string `json:"User-Agent"`
	WebKitVersion   string `json:"WebKit-Version"`
}

func NewCDPHealthChecker(port string) *CDPHealthChecker {
	return &CDPHealthChecker{
		port: port,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *CDPHealthChecker) Check() bool {
	url := fmt.Sprintf("http://127.0.0.1:%s/json/version", c.port)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var version CDPVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return false
	}

	return version.Browser != ""
}
