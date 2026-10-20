package app

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"git-notify/types/gitlab"
)

type gitlabAPIUser struct {
	Discord string `json:"discord"`
}

var (
	gitlabDiscordCache   = make(map[string]string)
	gitlabDiscordCacheMu sync.RWMutex
)

func gitLabDiscordID(user gitlab.User) string {
	cacheKey := gitlabUserCacheKey(user)
	if cacheKey == "" {
		return ""
	}

	gitlabDiscordCacheMu.RLock()
	if id, ok := gitlabDiscordCache[cacheKey]; ok {
		gitlabDiscordCacheMu.RUnlock()
		return id
	}
	gitlabDiscordCacheMu.RUnlock()

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		log.Println("GITLAB_TOKEN ไม่ได้ถูกตั้งค่า ข้ามการ lookup Discord ID")
		return ""
	}

	baseURL := strings.TrimRight(os.Getenv("GITLAB_URL"), "/")
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	var apiURL string
	if user.ID > 0 {
		apiURL = fmt.Sprintf("%s/api/v4/users/%d", baseURL, user.ID)
	} else {
		apiURL = fmt.Sprintf("%s/api/v4/users?username=%s", baseURL, url.QueryEscape(user.Username))
	}

	discordID := fetchGitLabDiscordID(apiURL, token)

	gitlabDiscordCacheMu.Lock()
	gitlabDiscordCache[cacheKey] = discordID
	gitlabDiscordCacheMu.Unlock()

	return discordID
}

func fetchGitLabDiscordID(apiURL, token string) string {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		log.Printf("GitLab API request error: %v", err)
		return ""
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		log.Printf("GitLab API call failed: %v", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitLab API read failed: %v", err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("GitLab API returned %d: %s", resp.StatusCode, body)
		return ""
	}

	if strings.Contains(apiURL, "username=") {
		var users []gitlabAPIUser
		if err := json.Unmarshal(body, &users); err != nil {
			log.Printf("GitLab API parse failed: %v", err)
			return ""
		}
		if len(users) == 0 {
			return ""
		}
		return strings.TrimSpace(users[0].Discord)
	}

	var user gitlabAPIUser
	if err := json.Unmarshal(body, &user); err != nil {
		log.Printf("GitLab API parse failed: %v", err)
		return ""
	}
	return strings.TrimSpace(user.Discord)
}

func gitlabUserCacheKey(user gitlab.User) string {
	if user.ID > 0 {
		return fmt.Sprintf("id:%d", user.ID)
	}
	username := strings.TrimSpace(user.Username)
	if username == "" {
		return ""
	}
	return "user:" + strings.ToLower(username)
}
