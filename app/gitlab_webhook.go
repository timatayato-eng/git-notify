package app

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"git-notify/types/gitlab"
)

type Notifier interface {
	Send(content string, mentionUserIDs []string) error
}

// discordIDLookup is swapped out in tests.
var discordIDLookup = gitLabDiscordID

func GitLabWebhookHandler(n Notifier, secret string, w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if secret != "" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Gitlab-Token")), []byte(secret)) != 1 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-Gitlab-Event")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[GitLab] event=%s body=%s", eventType, string(body))

	var payload gitlab.Payload
	if err := json.Unmarshal(body, &payload); err != nil {
		notify(n, fmt.Sprintf("**GitLab: %s**\n```json\n%s\n```", eventType, body), nil)
		w.WriteHeader(http.StatusOK)
		return
	}

	if assignees := newlyAssigned(payload); len(assignees) > 0 {
		log.Printf("[GitLab] new assignees: %s", formatUsers(assignees))
		notifyAssignEvent(n, payload, assignees)
	} else {
		log.Printf("[GitLab] no new assignees action=%s kind=%s", payload.ObjectAttributes.Action, payload.ObjectKind)
		logAssignDebug(payload)
		notify(n, formatGitLabMessage(eventType, payload), nil)
	}

	w.WriteHeader(http.StatusOK)
}

func notifyAssignEvent(n Notifier, payload gitlab.Payload, assignees []gitlab.User) {
	names := make([]string, len(assignees))
	var mentionIDs []string
	for i, assignee := range assignees {
		discordID := discordIDLookup(assignee)
		if discordID == "" {
			log.Printf("ไม่พบ Discord ID ใน GitLab profile สำหรับ: %s", assignee.Name)
			names[i] = assignee.Name
			continue
		}
		names[i] = fmt.Sprintf("<@%s>", discordID)
		mentionIDs = append(mentionIDs, discordID)
	}

	notify(n, formatAssignMessage(payload, "ให้ "+strings.Join(names, ", ")), mentionIDs)
}

func notify(n Notifier, message string, mentionIDs []string) {
	if err := n.Send(message, mentionIDs); err != nil {
		log.Printf("ไม่สามารถส่ง Discord notify: %v", err)
	}
}

func formatGitLabMessage(eventType string, payload gitlab.Payload) string {
	switch payload.ObjectKind {
	case "push":
		return fmt.Sprintf("**GitLab Push** `%s`\n%s push ไปที่ `%s`\n%s",
			payload.Project.Name, payload.UserName, payload.Ref, payload.Project.WebURL)
	case "merge_request", "issue", "work_item":
		kind := "MR"
		if payload.ObjectKind != "merge_request" {
			kind = "Issue"
		}
		return fmt.Sprintf("**GitLab %s** `%s` — %s\nโปรเจกต์: %s\n%s",
			kind, payload.ObjectAttributes.Action, payload.ObjectAttributes.Title,
			payload.Project.Name, payload.ObjectAttributes.URL)
	default:
		return fmt.Sprintf("**GitLab: %s**\nโปรเจกต์: %s", eventType, payload.Project.Name)
	}
}

func formatAssignMessage(payload gitlab.Payload, assigneeText string) string {
	return fmt.Sprintf("**GitLab Assign** — %s\n%s assign %s\nโปรเจกต์: %s\n%s",
		payload.ObjectAttributes.Title, assignerName(payload), assigneeText,
		payload.Project.Name, payload.ObjectAttributes.URL)
}

func newlyAssigned(payload gitlab.Payload) []gitlab.User {
	if change := payload.Changes.Assignees; change != nil {
		current := currentAssignees(payload)
		if len(current) == 0 {
			return nil
		}
		return filterNewAssignees(current, change.Previous)
	}
	if change := payload.Changes.AssigneeIDs; change != nil {
		return filterNewAssigneesByID(currentAssignees(payload), change.Previous)
	}
	if change := payload.Changes.AssigneeID; change != nil {
		var previousIDs []int
		if change.Previous != nil {
			previousIDs = []int{*change.Previous}
		}
		return filterNewAssigneesByID(currentAssignees(payload), previousIDs)
	}
	if payload.ObjectAttributes.Action == "open" && len(payload.Assignees) > 0 {
		return payload.Assignees
	}
	return nil
}

func filterNewAssigneesByID(current []gitlab.User, previousIDs []int) []gitlab.User {
	if len(current) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(previousIDs))
	for _, id := range previousIDs {
		seen[id] = true
	}
	var added []gitlab.User
	for _, user := range current {
		if user.ID == 0 || !seen[user.ID] {
			added = append(added, user)
		}
	}
	return added
}

func filterNewAssignees(current, previous []gitlab.User) []gitlab.User {
	seen := make(map[string]bool, len(previous))
	for _, user := range previous {
		seen[userKey(user)] = true
	}
	var added []gitlab.User
	for _, user := range current {
		if !seen[userKey(user)] {
			added = append(added, user)
		}
	}
	return added
}

func currentAssignees(payload gitlab.Payload) []gitlab.User {
	if len(payload.Assignees) > 0 {
		return payload.Assignees
	}
	if payload.Changes.Assignees != nil {
		return payload.Changes.Assignees.Current
	}
	return nil
}

func assignerName(payload gitlab.Payload) string {
	if payload.User.Name != "" {
		return payload.User.Name
	}
	return payload.UserName
}

func userKey(user gitlab.User) string {
	if user.ID > 0 {
		return fmt.Sprintf("id:%d", user.ID)
	}
	if user.Username != "" {
		return strings.ToLower(user.Username)
	}
	return strings.ToLower(user.Name)
}

func formatUsers(users []gitlab.User) string {
	names := make([]string, len(users))
	for i, u := range users {
		names[i] = u.Name
	}
	return strings.Join(names, ", ")
}

func logAssignDebug(payload gitlab.Payload) {
	if payload.ObjectAttributes.Action != "update" {
		return
	}
	ch := payload.Changes
	log.Printf("[GitLab] assign debug assignees=%d has_changes.assignees=%v has_changes.assignee_ids=%v has_changes.assignee_id=%v",
		len(payload.Assignees), ch.Assignees != nil, ch.AssigneeIDs != nil, ch.AssigneeID != nil)
}
