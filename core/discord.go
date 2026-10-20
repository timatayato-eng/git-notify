package core

import (
	"git-notify/app"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	Session *discordgo.Session
}

func InitDiscord() *DiscordBot {
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_TOKEN ไม่ได้ถูกตั้งค่า")
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Error creating Discord session: ", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds
	return &DiscordBot{Session: dg}
}

// RegisterCommands must be called after the session is open.
// With DISCORD_GUILD_ID the command appears immediately in that server only.
func (b *DiscordBot) RegisterCommands() {
	guildID := os.Getenv("DISCORD_GUILD_ID")
	if _, err := b.Session.ApplicationCommandBulkOverwrite(b.Session.State.User.ID, guildID, app.Commands); err != nil {
		log.Fatal("ลงทะเบียนคำสั่ง /git-notify ไม่สำเร็จ: ", err)
	}
	log.Println("ลงทะเบียนคำสั่ง /git-notify แล้ว")
}

func (b *DiscordBot) DM(discordUserID, message string) error {
	channel, err := b.Session.UserChannelCreate(discordUserID)
	if err != nil {
		return err
	}
	_, err = b.Session.ChannelMessageSend(channel.ID, message)
	return err
}
