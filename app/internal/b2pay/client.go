package b2pay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"umani-service/app/internal/config"
)

const (
	defaultBaseURL     = "https://app.b2pay.online"
	defaultTokenExpiry = 24
	tokenRefreshSlack  = 5 * time.Minute
)

type Client struct {
	httpClient *http.Client

	mu         sync.Mutex
	token      string
	validUntil time.Time
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) baseURL() string {
	cfg := config.GetConfig()
	u := strings.TrimSpace(cfg.B2PayBaseURL)
	if u == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(u, "/")
}

func (c *Client) tokenExpiryHours() int {
	h := config.GetConfig().B2PayTokenExpiryHours
	if h <= 0 {
		return defaultTokenExpiry
	}
	if h > 720 {
		return 720
	}
	return h
}

func (c *Client) getToken() (string, error) {
	cfg := config.GetConfig()
	if cfg.B2PayUserID == "" || cfg.B2PayEmail == "" || cfg.B2PayAPIKey == "" {
		return "", fmt.Errorf("b2pay: b2pay_user_id, b2pay_email, b2pay_api_key are required in config")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if c.token != "" && now.Before(c.validUntil) {
		return c.token, nil
	}

	body, err := json.Marshal(map[string]any{
		"user_id":            cfg.B2PayUserID,
		"email":              cfg.B2PayEmail,
		"api_key":            cfg.B2PayAPIKey,
		"token_expiry_hours": c.tokenExpiryHours(),
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL()+"/v1/auth/token/get", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("b2pay token: HTTP %d: %s", resp.StatusCode, string(data))
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("b2pay: empty token in response")
	}

	c.token = out.Token
	c.validUntil = now.Add(time.Duration(c.tokenExpiryHours())*time.Hour - tokenRefreshSlack)
	return c.token, nil
}

func (c *Client) invalidateToken() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = ""
	c.validUntil = time.Time{}
}

// CreateInvoice calls POST /v1/invoices. On 401, invalidates the token, obtains a new one, and retries once.
func (c *Client) CreateInvoice(payload []byte) ([]byte, int, error) {
	url := c.baseURL() + "/v1/invoices"
	var lastCode int
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.getToken()
		if err != nil {
			return nil, 0, err
		}
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, 0, err
		}
		body, rerr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if rerr != nil {
			return nil, resp.StatusCode, rerr
		}
		lastCode = resp.StatusCode
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			c.invalidateToken()
			continue
		}
		return body, resp.StatusCode, nil
	}
	return nil, lastCode, fmt.Errorf("b2pay: create invoice: unauthorized after retry")
}

// TransactionStatus returns GET /v1/transactions/{id}/status
func (c *Client) TransactionStatus(transactionID string) ([]byte, int, error) {
	tok, err := c.getToken()
	if err != nil {
		return nil, 0, err
	}
	url := fmt.Sprintf("%s/v1/transactions/%s/status", c.baseURL(), transactionID)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusUnauthorized {
		c.invalidateToken()
		tok2, err := c.getToken()
		if err != nil {
			return body, resp.StatusCode, err
		}
		req2, _ := http.NewRequest(http.MethodGet, url, nil)
		req2.Header.Set("Authorization", "Bearer "+tok2)
		resp2, err := c.httpClient.Do(req2)
		if err != nil {
			return body, resp.StatusCode, err
		}
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)
		return body2, resp2.StatusCode, nil
	}
	return body, resp.StatusCode, nil
}

// VerifyCallbackSignature checks X-Callback-Signature: sha256=<hex> per B2Pay docs.
func VerifyCallbackSignature(apiKey string, rawBody []byte, header string) bool {
	if apiKey == "" || len(rawBody) == 0 {
		return false
	}
	const prefix = "sha256="
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	wantHex := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(rawBody)
	sum := mac.Sum(nil)
	gotHex := hex.EncodeToString(sum)
	return hmac.Equal([]byte(wantHex), []byte(gotHex))
}
