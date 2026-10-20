package main

import (
	"git-notify/app"
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

	storePath := os.Getenv("USER_STORE_PATH")
	if storePath == "" {
		storePath = "data/users.json"
	}
	store, err := app.OpenUserStore(storePath)
	if err != nil {
		log.Fatal("เปิดไฟล์ข้อมูลผู้ใช้ไม่สำเร็จ: ", err)
	}

	bot := core.InitDiscord()
	bot.Session.AddHandler(app.CommandHandler(store))

	if err := bot.Session.Open(); err != nil {
		log.Fatal("ไม่สามารถเชื่อมต่อ Discord:", err)
	}
	defer bot.Session.Close()

	bot.RegisterCommands()

	core.StartHTTPServer(bot, store)

	log.Println("git-notify กำลังทำงาน... กด CTRL+C เพื่อหยุด")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
}
