package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"git-notify/types/github"
)

type Notifier interface {
	Send(content string, mentionUserIDs []string) error
}

type GitHubWebhook struct {
	Notifier Notifier
	Secret   string
	// DiscordUsers maps lowercase GitHub login to Discord user ID.
	DiscordUsers map[string]string
}

func (h *GitHubWebhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if h.Secret != "" && !validSignature(h.Secret, body, r.Header.Get("X-Hub-Signature-256")) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	log.Printf("[GitHub] event=%s body=%s", event, string(body))

	if event == "ping" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload github.Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON (ตั้ง Content type เป็น application/json)", http.StatusBadRequest)
		return
	}

	if payload.Action == "assigned" && payload.Assignee != nil {
		h.notifyAssign(payload)
	} else {
		h.notify(formatGitHubMessage(event, payload), nil)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *GitHubWebhook) notifyAssign(payload github.Payload) {
	login := payload.Assignee.Login
	assignee := login
	var mentionIDs []string
	if discordID := h.DiscordUsers[strings.ToLower(login)]; discordID != "" {
		assignee = fmt.Sprintf("<@%s>", discordID)
		mentionIDs = []string{discordID}
	} else {
		log.Printf("ไม่พบ Discord ID ใน DISCORD_USER_MAP สำหรับ: %s", login)
	}

	item := payloadItem(payload)
	h.notify(fmt.Sprintf("**GitHub Assign** — #%d %s\n%s assign ให้ %s\nRepo: %s\n%s",
		item.Number, item.Title, payload.Sender.Login, assignee,
		payload.Repository.FullName, item.HTMLURL), mentionIDs)
}

func (h *GitHubWebhook) notify(message string, mentionIDs []string) {
	if err := h.Notifier.Send(message, mentionIDs); err != nil {
		log.Printf("ไม่สามารถส่ง Discord notify: %v", err)
	}
}

func formatGitHubMessage(event string, payload github.Payload) string {
	switch event {
	case "push":
		branch := strings.TrimPrefix(strings.TrimPrefix(payload.Ref, "refs/heads/"), "refs/tags/")
		return fmt.Sprintf("**GitHub Push** `%s`\n%s push %d commit ไปที่ `%s`\n%s",
			payload.Repository.FullName, payload.Pusher.Name, len(payload.Commits), branch, payload.Compare)
	case "pull_request", "issues":
		kind := "PR"
		if event == "issues" {
			kind = "Issue"
		}
		item := payloadItem(payload)
		return fmt.Sprintf("**GitHub %s** `%s` — #%d %s\nโดย %s\nRepo: %s\n%s",
			kind, payload.Action, item.Number, item.Title, payload.Sender.Login,
			payload.Repository.FullName, item.HTMLURL)
	default:
		return fmt.Sprintf("**GitHub: %s** `%s`\nRepo: %s\n%s",
			event, payload.Action, payload.Repository.FullName, payload.Repository.HTMLURL)
	}
}

func payloadItem(payload github.Payload) github.Item {
	if payload.PullRequest != nil {
		return *payload.PullRequest
	}
	if payload.Issue != nil {
		return *payload.Issue
	}
	return github.Item{}
}

func validSignature(secret string, body []byte, header string) bool {
	got, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	gotBytes, err := hex.DecodeString(got)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hmac.Equal(gotBytes, mac.Sum(nil))
}

// ParseDiscordUserMap parses "githubLogin:discordID,otherLogin:otherID".
func ParseDiscordUserMap(raw string) map[string]string {
	users := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		login, id, ok := strings.Cut(strings.TrimSpace(pair), ":")
		login, id = strings.TrimSpace(login), strings.TrimSpace(id)
		if !ok || login == "" || id == "" {
			if pair = strings.TrimSpace(pair); pair != "" {
				log.Printf("DISCORD_USER_MAP รูปแบบไม่ถูกต้อง ข้าม: %q", pair)
			}
			continue
		}
		users[strings.ToLower(login)] = id
	}
	return users
}
