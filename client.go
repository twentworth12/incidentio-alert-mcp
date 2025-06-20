package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type IncidentIOClient struct {
	webhookURL string
	apiToken   string
}

func NewIncidentIOClient() *IncidentIOClient {
	webhookURL := os.Getenv("INCIDENTIO_WEBHOOK_URL")
	if webhookURL == "" {
		panic("INCIDENTIO_WEBHOOK_URL environment variable is required")
	}
	
	apiToken := os.Getenv("INCIDENTIO_API_TOKEN")
	if apiToken == "" {
		panic("INCIDENTIO_API_TOKEN environment variable is required")
	}
	
	return &IncidentIOClient{
		webhookURL: webhookURL,
		apiToken:   apiToken,
	}
}

type AlertEvent struct {
	Title            string                 `json:"title"`
	Description      string                 `json:"description,omitempty"`
	DeduplicationKey string                 `json:"deduplication_key"`
	Status           string                 `json:"status"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

func (c *IncidentIOClient) SendAlert(alert AlertEvent) error {
	payload, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}
	
	req, err := http.NewRequest("POST", c.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}