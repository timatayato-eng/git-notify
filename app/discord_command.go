package app

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const linkCommandName = "git-notify"

var githubLoginPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})$`)

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        linkCommandName,
		Description: "ผูกบัญชี GitHub เพื่อรับแจ้งเตือนทาง DM",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "github",
				Description: "ชื่อผู้ใช้ GitHub เช่น octocat",
				Required:    true,
			},
		},
	},
}

func CommandHandler(store *UserStore) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}
		data := i.ApplicationCommandData()
		if data.Name != linkCommandName || len(data.Options) == 0 {
			return
		}

		user := i.User
		if i.Member != nil {
			user = i.Member.User
		}

		reply(s, i, linkGitHub(store, user.ID, data.Options[0].StringValue()))
	}
}

func linkGitHub(store *UserStore, discordID, input string) string {
	login, ok := normalizeGitHubLogin(input)
	if !ok {
		return fmt.Sprintf("ชื่อ GitHub `%s` ไม่ถูกต้อง ลองใหม่อีกครั้ง เช่น `/git-notify github:octocat`", input)
	}
	if err := store.Link(login, discordID); err != nil {
		log.Printf("บันทึกการผูกบัญชีล้มเหลว %s -> %s: %v", login, discordID, err)
		return "บันทึกไม่สำเร็จ กรุณาลองใหม่ภายหลัง"
	}
	log.Printf("ผูกบัญชี GitHub %s กับ Discord %s", login, discordID)
	return fmt.Sprintf("ผูกกับ GitHub `%s` แล้ว เมื่อมีการ assign งานหรืออัปเดต issue ของคุณ จะได้รับแจ้งเตือนทาง DM", login)
}

// normalizeGitHubLogin accepts "octocat", "@octocat" or "https://github.com/octocat".
func normalizeGitHubLogin(input string) (string, bool) {
	login := strings.TrimSpace(input)
	login = strings.TrimPrefix(login, "https://")
	login = strings.TrimPrefix(login, "github.com/")
	login = strings.TrimPrefix(login, "@")
	login = strings.TrimSuffix(login, "/")
	return login, githubLoginPattern.MatchString(login)
}

func reply(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("ตอบกลับคำสั่งไม่สำเร็จ: %v", err)
	}
}
