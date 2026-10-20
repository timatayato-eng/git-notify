package app

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// UserStore persists GitHub login -> Discord user ID links in a JSON file.
type UserStore struct {
	path  string
	mu    sync.RWMutex
	users map[string]string
}

func OpenUserStore(path string) (*UserStore, error) {
	s := &UserStore{path: path, users: make(map[string]string)}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &s.users); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *UserStore) DiscordID(githubLogin string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[strings.ToLower(githubLogin)]
}

// Link maps githubLogin to discordID, replacing any login the Discord user linked before.
func (s *UserStore) Link(githubLogin, discordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for login, id := range s.users {
		if id == discordID {
			delete(s.users, login)
		}
	}
	s.users[strings.ToLower(githubLogin)] = discordID
	return s.save()
}

func (s *UserStore) save() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
