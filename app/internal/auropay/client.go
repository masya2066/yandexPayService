package auropay

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
	"umani-service/app/internal/config"
)

const defaultBaseURL = "https://app.aurapay.tech"

// Client — HTTP-клиент Aurapay (заголовки X-ApiKey, X-ShopId).
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) baseURL() string {
	u := strings.TrimSpace(config.GetConfig().AuropayBaseURL)
	if u == "" {
		return defaultBaseURL
	}
	return strings.TrimRight(u, "/")
}

func (c *Client) headers() (http.Header, error) {
	cfg := config.GetConfig()
	apiKey := strings.TrimSpace(cfg.AuropayAPIKey)
	shopID := strings.TrimSpace(cfg.AuropayShopID)
	if apiKey == "" || shopID == "" {
		return nil, fmt.Errorf("auropay: auropay_api_key и auropay_shop_id обязательны в config")
	}
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	h.Set("X-ApiKey", apiKey)
	h.Set("X-ShopId", shopID)
	return h, nil
}

// CreateInvoice вызывает POST /invoice/create.
func (c *Client) CreateInvoice(payload []byte) ([]byte, int, error) {
	h, err := c.headers()
	if err != nil {
		return nil, 0, err
	}
	url := c.baseURL() + "/invoice/create"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, err
	}
	req.Header = h
	log.Printf("[auropay] POST /invoice/create исходящее тело: %s", string(payload))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	log.Printf("[auropay] POST /invoice/create ответ HTTP %d тело: %s", resp.StatusCode, string(body))
	return body, resp.StatusCode, nil
}

// InvoiceStatus вызывает POST /invoice/status (тело: {"order_id":...} или {"id":...}).
func (c *Client) InvoiceStatus(statusBody []byte) ([]byte, int, error) {
	h, err := c.headers()
	if err != nil {
		return nil, 0, err
	}
	url := c.baseURL() + "/invoice/status"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(statusBody))
	if err != nil {
		return nil, 0, err
	}
	req.Header = h
	log.Printf("[auropay] POST /invoice/status исходящее тело: %s", string(statusBody))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	log.Printf("[auropay] POST /invoice/status ответ HTTP %d тело: %s", resp.StatusCode, string(body))
	return body, resp.StatusCode, nil
}

// VerifyWebhookSignature проверяет X-SIGNATURE: HMAC-SHA256(secret, concat(sorted values))) в hex.
func VerifyWebhookSignature(secret string, params map[string]any, receivedHex string) bool {
	secret = strings.TrimSpace(secret)
	receivedHex = strings.TrimSpace(strings.ToLower(receivedHex))
	if secret == "" || receivedHex == "" || len(params) == 0 {
		return false
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var concat strings.Builder
	for _, k := range keys {
		concat.WriteString(valueForSignature(params[k]))
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(concat.String()))
	want := hex.EncodeToString(mac.Sum(nil))
	wantB := []byte(want)
	gotB := []byte(receivedHex)
	if len(wantB) != len(gotB) {
		return false
	}
	return hmac.Equal(wantB, gotB)
}

func valueForSignature(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// JSON без кавычек; стараемся не добавлять лишних нулей
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", t), "0"), ".")
	case bool:
		if t {
			return "true"
		}
		return "false"
	case json.Number:
		return t.String()
	default:
		return fmt.Sprint(t)
	}
}

// IsPayoutWebhook — webhook выплаты (в теле есть amount_to_payout; у инвойса этого поля нет).
func IsPayoutWebhook(params map[string]any) bool {
	_, ok := params["amount_to_payout"]
	return ok
}
