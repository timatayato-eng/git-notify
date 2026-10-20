package main

import (
	"git-notify/core"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("ไม่พบไฟล์ .env ใช้ environment variables จากระบบแทน")
	}

	discord := core.InitDiscordWebhook()

	core.StartHTTPServer(discord)

	log.Println("git-notify กำลังทำงาน... กด CTRL+C เพื่อหยุด")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
