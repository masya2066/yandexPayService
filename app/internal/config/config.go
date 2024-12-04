package config

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"sync"
)

type Config struct {
	Receiver   string `json:"receiver"`    // yandex_number_wallet
	SuccessURL string `json:"success_url"` // success url to send user after success payment
	FailURL    string `json:"fail_url"`    // failed url to send user after failed payment
	SendURL    string `json:"send_url"`    // url to send notification to another service
	AppPort    string `json:"app_port"`    // service port
}

var (
	currentConfig Config
	mutex         sync.RWMutex
	configPath    = ""
)

func LoadConfig() Config {
	mutex.Lock()
	defer mutex.Unlock()

	if err := os.Getenv("CONFIG_PATH"); err != "" {
		slog.Default().Error("CONFIG_PATH is required! CONFIG_PATH is set to %s", err)
	}

	file, err := os.Open(configPath)
	if err != nil {
		// Create file if not exist
		currentConfig = Config{
			Receiver:   "yandex_wallet_number",
			SuccessURL: "http://localhost:8080/success",
			FailURL:    "http://localhost:8080/fail",
			SendURL:    "http://localhost:8080/send",
			AppPort:    "8080",
		}
		slog.Default().Info("Config file not found. Creating default config...")
		saveStartConfig(currentConfig)
		return currentConfig
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&currentConfig); err != nil {
		panic("Failed to decode config: " + err.Error())
	}

	return currentConfig
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
