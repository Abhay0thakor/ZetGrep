package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookPayload struct {
	Content string `json:"content,omitempty"` // Discord
	Text    string `json:"text,omitempty"`    // Slack
}

func SendWebhook(url, wType, message string) error {
	if url == "" {
		return nil
	}

	payload := WebhookPayload{}
	switch wType {
	case "discord":
		payload.Content = message
	case "slack":
		payload.Text = message
	default:
		payload.Text = message
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status: %d", resp.StatusCode)
	}

	return nil
}
