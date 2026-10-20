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

	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret == "" {
		log.Println("GITHUB_WEBHOOK_SECRET ไม่ได้ถูกตั้งค่า webhook จะรับ request จากทุกที่")
	}

	http.Handle("/webhook/github", &app.GitHubWebhook{
		Notifier:     notifier,
		Secret:       secret,
		DiscordUsers: app.ParseDiscordUserMap(os.Getenv("DISCORD_USER_MAP")),
	})

	go func() {
		log.Printf("GitHub webhook server กำลังทำงานที่ port %s", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()
}
