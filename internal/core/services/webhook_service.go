package services

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "go.uber.org/zap"
    "E.E/internal/core/domain"
)

type WebhookService struct {
    logger     *zap.Logger
    httpClient *http.Client
    configs    map[string]domain.WebhookConfig
}

func NewWebhookService(logger *zap.Logger) *WebhookService {
    return &WebhookService{
        logger:     logger,
        httpClient: &http.Client{Timeout: 10 * time.Second},
        configs:    make(map[string]domain.WebhookConfig),
    }
}

func (s *WebhookService) RegisterWebhook(config domain.WebhookConfig) error {
    if config.URL == "" {
        return fmt.Errorf("webhook URL is required")
    }
    if config.Secret == "" {
        return fmt.Errorf("webhook secret is required")
    }
    
    s.configs[config.URL] = config
    return nil
}

func (s *WebhookService) SendWebhook(payload domain.WebhookPayload, config domain.WebhookConfig) error {
    // Sign payload
    payload.Signature = s.signPayload(payload, config.Secret)

    // Marshal payload
    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal webhook payload: %w", err)
    }

    // Create request
    req, err := http.NewRequest(http.MethodPost, config.URL, bytes.NewBuffer(jsonPayload))
    if err != nil {
        return fmt.Errorf("failed to create webhook request: %w", err)
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-Webhook-Signature", payload.Signature)

    // Send request
    resp, err := s.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to send webhook: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 300 {
        return fmt.Errorf("webhook failed with status: %d", resp.StatusCode)
    }

    return nil
}

// signPayload creates an HMAC SHA256 signature of the payload
func (s *WebhookService) signPayload(payload domain.WebhookPayload, secret string) string {
    // Create a copy of payload without the signature
    payloadCopy := payload
    payloadCopy.Signature = ""

    data, err := json.Marshal(payloadCopy)
    if err != nil {
        s.logger.Error("Failed to marshal payload for signing", zap.Error(err))
        return ""
    }

    h := hmac.New(sha256.New, []byte(secret))
    h.Write(data)
    return hex.EncodeToString(h.Sum(nil))
}