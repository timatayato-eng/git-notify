package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const discordMaxContentLength = 2000

type DiscordWebhook struct {
	URL    string
	Client *http.Client
}

type allowedMentions struct {
	Parse []string `json:"parse"`
	Users []string `json:"users"`
}

type webhookMessage struct {
	Content         string          `json:"content"`
	Username        string          `json:"username,omitempty"`
	AllowedMentions allowedMentions `json:"allowed_mentions"`
}

func InitDiscordWebhook() *DiscordWebhook {
	url := os.Getenv("DISCORD_WEBHOOK_URL")
	if url == "" {
		log.Fatal("DISCORD_WEBHOOK_URL ไม่ได้ถูกตั้งค่า")
	}
	return &DiscordWebhook{URL: url, Client: &http.Client{Timeout: 10 * time.Second}}
}

// Send posts content to the channel; only users listed in mentionUserIDs are pinged.
func (d *DiscordWebhook) Send(content string, mentionUserIDs []string) error {
	if mentionUserIDs == nil {
		mentionUserIDs = []string{}
	}
	body, err := json.Marshal(webhookMessage{
		Content:         truncate(content, discordMaxContentLength),
		Username:        os.Getenv("DISCORD_WEBHOOK_USERNAME"),
		AllowedMentions: allowedMentions{Parse: []string{}, Users: mentionUserIDs},
	})
	if err != nil {
		return err
	}

	resp, err := d.Client.Post(d.URL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Discord webhook ตอบกลับ %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
