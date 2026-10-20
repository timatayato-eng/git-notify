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

const DefaultNotifyActions = "assigned,closed,reopened,labeled,unlabeled"

type DirectMessenger interface {
	DM(discordUserID, message string) error
}

type DiscordLookup interface {
	DiscordID(githubLogin string) string
}

type GitHubWebhook struct {
	Messenger DirectMessenger
	Users     DiscordLookup
	Secret    string
	Actions   map[string]bool
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
	if event == "ping" {
		log.Println("[GitHub] ping สำเร็จ")
		w.WriteHeader(http.StatusOK)
		return
	}

	var payload github.Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON (ตั้ง Content type เป็น application/json)", http.StatusBadRequest)
		return
	}

	item, kind := payloadItem(event, payload)
	if item == nil || !h.Actions[payload.Action] {
		log.Printf("[GitHub] ข้าม event=%s action=%s", event, payload.Action)
		w.WriteHeader(http.StatusOK)
		return
	}

	message := formatMessage(kind, *item, payload)
	for _, login := range recipients(*item, payload) {
		h.sendDM(login, message)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *GitHubWebhook) sendDM(login, message string) {
	discordID := h.Users.DiscordID(login)
	if discordID == "" {
		log.Printf("[GitHub] %s ยังไม่ได้ผูกบัญชีด้วย /git-notify ข้ามการส่ง DM", login)
		return
	}
	if err := h.Messenger.DM(discordID, message); err != nil {
		log.Printf("ส่ง DM ไปยัง %s (%s) ไม่สำเร็จ: %v", login, discordID, err)
		return
	}
	log.Printf("ส่ง DM ไปยัง %s (%s) สำเร็จ", login, discordID)
}

func payloadItem(event string, payload github.Payload) (*github.Item, string) {
	switch event {
	case "issues":
		return payload.Issue, "Issue"
	case "pull_request":
		return payload.PullRequest, "PR"
	}
	return nil, ""
}

// recipients returns who should get a DM, never including the person who made the change.
func recipients(item github.Item, payload github.Payload) []string {
	candidates := item.Assignees
	if payload.Action == "assigned" || payload.Action == "unassigned" {
		candidates = nil
		if payload.Assignee != nil {
			candidates = []github.User{*payload.Assignee}
		}
	}

	seen := map[string]bool{strings.ToLower(payload.Sender.Login): true}
	var logins []string
	for _, u := range candidates {
		key := strings.ToLower(u.Login)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		logins = append(logins, u.Login)
	}
	return logins
}

func formatMessage(kind string, item github.Item, payload github.Payload) string {
	return fmt.Sprintf("**[%s] %s #%d** %s\n%s\n%s",
		payload.Repository.FullName, kind, item.Number, item.Title,
		describeAction(payload, item), item.HTMLURL)
}

func describeAction(payload github.Payload, item github.Item) string {
	sender := payload.Sender.Login
	label := ""
	if payload.Label != nil {
		label = payload.Label.Name
	}
	switch payload.Action {
	case "assigned":
		return sender + " assign งานนี้ให้คุณ"
	case "unassigned":
		return sender + " เอาคุณออกจากงานนี้"
	case "closed":
		if item.Merged {
			return sender + " merge แล้ว"
		}
		return sender + " ปิดแล้ว"
	case "reopened":
		return sender + " เปิดใหม่อีกครั้ง"
	case "labeled":
		return fmt.Sprintf("%s เพิ่ม label `%s`", sender, label)
	case "unlabeled":
		return fmt.Sprintf("%s ลบ label `%s`", sender, label)
	default:
		return fmt.Sprintf("%s อัปเดต: `%s`", sender, payload.Action)
	}
}

// ParseActions parses a comma-separated list such as "assigned,closed".
func ParseActions(raw string) map[string]bool {
	if strings.TrimSpace(raw) == "" {
		raw = DefaultNotifyActions
	}
	actions := make(map[string]bool)
	for _, a := range strings.Split(raw, ",") {
		if a = strings.ToLower(strings.TrimSpace(a)); a != "" {
			actions[a] = true
		}
	}
	return actions
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
