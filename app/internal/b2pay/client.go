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
	"umani-service/app/internal/models"
)

const defaultBaseURL = "https://app.b2pay.online"

type tokenGetRequest struct {
	UserID           string `json:"user_id"`
	Email            string `json:"email"`
	APIKey           string `json:"api_key"`
	TokenExpiryHours int    `json:"token_expiry_hours"`
}

type tokenGetResponse struct {
	Token string `json:"token"`
}

// InvoiceCreateRequest — тело POST /v1/invoices (объявлено в client.go, заполняется в handlers/b2pay.go).
type InvoiceCreateRequest struct {
	CustomerID          string         `json:"customer_id"`
	Amount              float64        `json:"amount"`
	Currency            string         `json:"currency"`
	Description         string         `json:"description"`
	IsReturningCustomer *bool          `json:"is_returning_customer,omitempty"`
	Metadata            map[string]any `json:"metadata"`
}

var (
	tokenMu        sync.Mutex
	lastCredFP     string
	cachedToken    string
	tokenDeadline  time.Time
)

func credFingerprint(cfg config.Config) string {
	return cfg.B2PayUserID + "\x00" + cfg.B2PayEmail + "\x00" + cfg.B2PayAPIKey
}

func baseURL(cfg config.Config) string {
	u := strings.TrimSpace(cfg.B2PayBaseURL)
	if u == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(u, "/")
}

func tokenExpiryHours(cfg config.Config) int {
	if cfg.B2PayTokenExpiryHours < 1 {
		return 24
	}
	if cfg.B2PayTokenExpiryHours > 720 {
		return 720
	}
	return cfg.B2PayTokenExpiryHours
}

// InvalidateTokenCache сбрасывает кэш JWT (после смены ключа или 401).
func InvalidateTokenCache() {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	cachedToken = ""
	lastCredFP = ""
	tokenDeadline = time.Time{}
}

func bearerToken(cfg config.Config) (string, error) {
	if cfg.B2PayUserID == "" || cfg.B2PayEmail == "" || cfg.B2PayAPIKey == "" {
		return "", fmt.Errorf("b2pay: не заданы user_id, email или api_key")
	}
	fp := credFingerprint(cfg)
	margin := 5 * time.Minute
	ttl := time.Duration(tokenExpiryHours(cfg)) * time.Hour

	tokenMu.Lock()
	defer tokenMu.Unlock()

	if cachedToken != "" && fp == lastCredFP && time.Now().Before(tokenDeadline) {
		return cachedToken, nil
	}

	body, err := json.Marshal(tokenGetRequest{
		UserID:           cfg.B2PayUserID,
		Email:            cfg.B2PayEmail,
		APIKey:           cfg.B2PayAPIKey,
		TokenExpiryHours: tokenExpiryHours(cfg),
	})
	if err != nil {
		return "", err
	}
	tok, err := postTokenJSON(cfg, baseURL(cfg)+"/v1/auth/token/get", body)
	if err != nil {
		return "", err
	}
	cachedToken = tok
	lastCredFP = fp
	tokenDeadline = time.Now().Add(ttl - margin)
	return cachedToken, nil
}

func postTokenJSON(cfg config.Config, url string, body []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("b2pay token: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var tr tokenGetResponse
	if err := json.Unmarshal(raw, &tr); err != nil {
		return "", err
	}
	if tr.Token == "" {
		return "", fmt.Errorf("b2pay token: пустой token в ответе")
	}
	return tr.Token, nil
}

// CreateInvoice создаёт счёт в B2Pay (POST /v1/invoices). При 401 один раз сбрасывает токен и повторяет запрос.
func CreateInvoice(cfg config.Config, inv InvoiceCreateRequest) (*models.B2PayInvoiceResponse, error) {
	do := func(bearer string) (*http.Response, []byte, error) {
		body, err := json.Marshal(inv)
		if err != nil {
			return nil, nil, err
		}
		req, err := http.NewRequest(http.MethodPost, baseURL(cfg)+"/v1/invoices", bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bearer)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, raw, nil
	}

	tok, err := bearerToken(cfg)
	if err != nil {
		return nil, err
	}
	resp, raw, err := do(tok)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		InvalidateTokenCache()
		tok2, err2 := bearerToken(cfg)
		if err2 != nil {
			return nil, err2
		}
		resp, raw, err = do(tok2)
		if err != nil {
			return nil, err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("b2pay invoices: HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out models.B2PayInvoiceResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifyCallbackSignature проверяет заголовок X-Callback-Signature для сырого JSON-тела.
func VerifyCallbackSignature(rawBody []byte, signatureHeader, apiKey string) bool {
	if apiKey == "" || signatureHeader == "" {
		return false
	}
	h := strings.TrimSpace(signatureHeader)
	eq := strings.Index(h, "=")
	if eq < 0 {
		return false
	}
	algo := strings.ToLower(strings.TrimSpace(h[:eq]))
	if algo != "sha256" {
		return false
	}
	hexPart := strings.TrimSpace(h[eq+1:])
	wantBytes, err := hex.DecodeString(hexPart)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(apiKey))
	mac.Write(rawBody)
	return hmac.Equal(mac.Sum(nil), wantBytes)
}
