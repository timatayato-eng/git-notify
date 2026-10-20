package core

import (
	"git-notify/app"
	"log"
	"net/http"
	"os"
)

func StartHTTPServer(notifier app.Notifier) {
	port := os.Getenv("WEBHOOK_PORT")
	if port == "" {
		port = "8080"
	}

	secret := os.Getenv("GITLAB_WEBHOOK_SECRET")
	if secret == "" {
		log.Println("GITLAB_WEBHOOK_SECRET ไม่ได้ถูกตั้งค่า webhook จะรับ request จากทุกที่")
	}

	http.HandleFunc("/webhook/gitlab", func(w http.ResponseWriter, r *http.Request) {
		app.GitLabWebhookHandler(notifier, secret, w, r)
	})

	go func() {
		log.Printf("GitLab webhook server กำลังทำงานที่ port %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()
}
