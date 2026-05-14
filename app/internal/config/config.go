package config

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Config struct {
	Receiver          string `json:"receiver"`             // yandex_number_wallet
	SecretWord        string `json:"secret_word"`          // secret_word
	AuthTokenCardLink string `json:"auth_token_card_link"` // auth token for card link
	ShopIDCardLink    string `json:"shop_id_card_link"`    // shop id
	SuccessURL        string `json:"success_url"`          // success url to send user after success payment
	FailURL           string `json:"fail_url"`             // failed url to send user after failed payment
	SendURL           string `json:"send_url"`             // url to send notification to another service
	AppPort           string `json:"app_port"`             // service port

	// B2Pay (https://app.b2pay.online) — see b2pay.md
	B2PayBaseURL          string `json:"b2pay_base_url"`           // default: https://app.b2pay.online
	B2PayUserID           string `json:"b2pay_user_id"`            // User ID in B2Pay
	B2PayEmail            string `json:"b2pay_email"`              // account email
	B2PayAPIKey           string `json:"b2pay_api_key"`            // API key (also used for callback HMAC)
	B2PayTokenExpiryHours int    `json:"b2pay_token_expiry_hours"` // 1–720, default 24
	B2PayTestMode         bool   `json:"b2pay_test_mode"`          // metadata.test_mode
	B2PayReturnURL        string `json:"b2pay_return_url"`         // user redirect after payment; default: success_url
	B2PayNotificationURL  string `json:"b2pay_notification_url"`   // public URL of POST /b2pay/order/notification

	// Aurapay (https://app.aurapay.tech) — см. auropay.md
	AuropayBaseURL          string `json:"auropay_base_url"`           // по умолчанию https://app.aurapay.tech
	AuropayAPIKey           string `json:"auropay_api_key"`            // заголовок X-ApiKey
	AuropayShopID           string `json:"auropay_shop_id"`            // заголовок X-ShopId
	AuropayWebhookSecret    string `json:"auropay_webhook_secret"`     // секретный ключ #2 для X-SIGNATURE webhook
	AuropayNotificationURL  string `json:"auropay_notification_url"` // публичный URL POST /auropay/order/notification
}

var (
	currentConfig Config
	mutex         sync.RWMutex
	configPath    = ""
)

// resolveConfigPath: CONFIG_PATH, else ./config.json if it exists, else config.json next to the executable, else "config.json" (CWD).
// This fixes 412 when the binary is run not from the project root but umani-service and config.json sit in the same folder.
func resolveConfigPath() string {
	// .env often sets CONFIG_PATH to another file; that file may miss B2Pay keys — we merge from CWD later.
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		return p
	}
	if _, err := os.Stat("config.json"); err == nil {
		if abs, err := filepath.Abs("config.json"); err == nil {
			return abs
		}
		return "config.json"
	}
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		next := filepath.Join(exeDir, "config.json")
		if _, err := os.Stat(next); err == nil {
			return next
		}
	}
	if abs, err := filepath.Abs("config.json"); err == nil {
		return abs
	}
	return "config.json"
}

func LoadConfig() Config {
	mutex.Lock()
	defer mutex.Unlock()

	configPath = resolveConfigPath()

	SQLitePath := os.Getenv("SQLITE_PATH")
	if SQLitePath == "" {
		os.Setenv("SQLITE_PATH", "unsentNotify.db")
	}

	file, err := os.Open(configPath)
	if err != nil {
		// Create file if not exist
		currentConfig = Config{
			Receiver:              "yandex_wallet_number",
			SecretWord:            "secret_word",
			AuthTokenCardLink:     "auth_token_card_link",
			ShopIDCardLink:        "shop_id_card_link",
			SuccessURL:            "http://localhost:8080/success",
			FailURL:               "http://localhost:8080/fail",
			SendURL:               "http://localhost:8080/send",
			AppPort:               "8080",
			B2PayBaseURL:          "https://app.b2pay.online",
			B2PayTokenExpiryHours: 24,
		}
		slog.Default().Info("Config file not found. Creating default config...", "path", configPath)
		saveStartConfig(currentConfig)
		return currentConfig
	}
	defer file.Close()

	currentConfig = Config{}
	if err := json.NewDecoder(file).Decode(&currentConfig); err != nil {
		panic("Failed to decode config: " + err.Error())
	}

	mergeB2PayFromCwdIfMissing()
	applyB2PayEnvOverrides()
	currentConfig.B2PayNotificationURL = strings.TrimSpace(currentConfig.B2PayNotificationURL)

	mergeAuropayFromCwdIfMissing()
	applyAuropayEnvOverrides()
	currentConfig.AuropayNotificationURL = strings.TrimSpace(currentConfig.AuropayNotificationURL)

	slog.Default().Info("Config loaded",
		"path", configPath,
		"env_CONFIG_PATH", os.Getenv("CONFIG_PATH"),
		"b2pay_notification_set", currentConfig.B2PayNotificationURL != "",
		"auropay_notification_set", currentConfig.AuropayNotificationURL != "")

	return currentConfig
}

// If CONFIG_PATH pointed to a file without B2Pay fields, use ./config.json from CWD when it has the URL.
func mergeB2PayFromCwdIfMissing() {
	if strings.TrimSpace(currentConfig.B2PayNotificationURL) != "" {
		return
	}
	cwdFile, err := filepath.Abs("config.json")
	if err != nil {
		return
	}
	primary, err := filepath.Abs(configPath)
	if err == nil && filepath.Clean(cwdFile) == filepath.Clean(primary) {
		return
	}
	f, err := os.Open("config.json")
	if err != nil {
		return
	}
	defer f.Close()
	var alt Config
	if err := json.NewDecoder(f).Decode(&alt); err != nil {
		return
	}
	if s := strings.TrimSpace(alt.B2PayNotificationURL); s != "" {
		slog.Default().Info("b2pay_notification_url merged from CWD config.json (main config file had it empty)", "main", configPath, "value", s)
		currentConfig.B2PayNotificationURL = s
	}
}

func applyB2PayEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv("B2PAY_NOTIFICATION_URL")); v != "" {
		currentConfig.B2PayNotificationURL = v
		slog.Default().Info("b2pay_notification_url set from B2PAY_NOTIFICATION_URL")
	}
}

// mergeAuropayFromCwdIfMissing — если в основном config нет auropay_notification_url, подставляем из ./config.json (как у B2Pay).
func mergeAuropayFromCwdIfMissing() {
	if strings.TrimSpace(currentConfig.AuropayNotificationURL) != "" {
		return
	}
	cwdFile, err := filepath.Abs("config.json")
	if err != nil {
		return
	}
	primary, err := filepath.Abs(configPath)
	if err == nil && filepath.Clean(cwdFile) == filepath.Clean(primary) {
		return
	}
	f, err := os.Open("config.json")
	if err != nil {
		return
	}
	defer f.Close()
	var alt Config
	if err := json.NewDecoder(f).Decode(&alt); err != nil {
		return
	}
	if s := strings.TrimSpace(alt.AuropayNotificationURL); s != "" {
		slog.Default().Info("auropay_notification_url merged from CWD config.json", "main", configPath)
		currentConfig.AuropayNotificationURL = s
	}
}

func applyAuropayEnvOverrides() {
	if v := strings.TrimSpace(os.Getenv("AUROPAY_NOTIFICATION_URL")); v != "" {
		currentConfig.AuropayNotificationURL = v
		slog.Default().Info("auropay_notification_url set from AUROPAY_NOTIFICATION_URL")
	}
}

// SaveConfig save config to JSON
func SaveConfig(cfg Config) {
	log.Println("Attempting to acquire lock for SaveConfig")
	mutex.Lock()
	log.Println("Lock acquired for SaveConfig")
	defer func() {
		mutex.Unlock()
		log.Println("Lock released for SaveConfig")
	}()

	file, err := os.Create(configPath)
	if err != nil {
		log.Fatalf("Failed to save config: %s", err.Error())
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		log.Fatalf("Failed to encode config: %s", err.Error())
	}

	currentConfig = cfg
	log.Println("Config saved successfully")
}

func saveStartConfig(cfg Config) {
	file, err := os.Create(configPath)
	if err != nil {
		log.Fatalf("Failed to save config: %s", err.Error())
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		log.Fatalf("Failed to encode config: %s", err.Error())
	}

	currentConfig = cfg
	log.Println("Config saved successfully")
}

func GetConfig() Config {
	mutex.RLock()
	defer mutex.RUnlock()
	return currentConfig
}
