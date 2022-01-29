package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var dmMessage string

func main() {
	dmMessage = os.Getenv("DM_MESSAGE")
	botToken := os.Getenv("BOT_TOKEN")

	discord, err := discordgo.New("Bot " + botToken)
	if err != nil {
		panic(err)
	}
	discord.AddHandler(filterNewMessage)

	err = discord.Open()
	if err != nil {
		panic(err)
	}

	log.Println("Filtering previous messages...")
	filterPreviousMessages(discord)
	log.Println("Filtering previous messages done.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Closing discord connection")

	discord.Close()
}

func containsWord(message string, words []string) bool {
	for _, word := range words {
		if strings.Contains(message, word) {
			return true
		}
	}
	return true
}

func containsSpamWords(message string) bool {
	messageLc := strings.ToLower(message)
	if strings.Contains(messageLc, "nitro") {
		blacklistedWords := []string{"airdrop", "free", "share your screen", "gift",
			"discord", "https", "everyone", "steam", "https://"}
		if containsWord(messageLc, blacklistedWords) {
			return true
		}
	}
	if strings.Contains(messageLc, "gift") {
		blacklistedWords := []string{"https://", "everyone", "gift?)", ".gift/"}
		if containsWord(messageLc, blacklistedWords) {
			return true
		}
	}
	return false
}

func isMessageASpam(m *discordgo.Message) bool {
	if containsSpamWords(m.Content) {
		return true
	}

	for _, embed := range m.Embeds {
		if containsSpamWords(embed.Title) || containsSpamWords(embed.Description) {
			return true
		}
	}
	return false
}

func filterMessage(s *discordgo.Session, m *discordgo.Message) {
	if !isMessageASpam(m) {
		return
	}

	err := s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		log.Printf("Could not delete message %s: %s\n", m.ID, err)
	}

	informUserAboutSpamRemoval(s, m.Author.ID)
}

func filterPreviousMessages(s *discordgo.Session) {
	for _, guild := range s.State.Guilds {
		channels, err := s.GuildChannels(guild.ID)
		if err != nil {
			panic(err)
		}

		for _, channel := range channels {
			messages, err := s.ChannelMessages(channel.ID, 100, "", "", "")
			if err != nil {
				panic(err)
			}
			for _, message := range messages {
				filterMessage(s, message)
			}
		}
	}
}

func filterNewMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if !isMessageASpam(m.Message) {
		return
	}

	log.Println("Deleting message:", m.ChannelID, m.Author.Username, m.Content, m.ID)
	err := s.ChannelMessageDelete(m.ChannelID, m.ID)
	if err != nil {
		log.Printf("Could not delete message %s: %s\n", m.ID, err)
	}

	informUserAboutSpamRemoval(s, m.Author.ID)
}

func informUserAboutSpamRemoval(s *discordgo.Session, authorID string) {
	if dmMessage == "" {
		return
	}

	channel, err := s.UserChannelCreate(authorID)
	if err != nil {
		log.Printf("Could not create dm channel to %s: %s\n", authorID, err)
		return
	}
	_, err = s.ChannelMessageSend(channel.ID, dmMessage)
	if err != nil {
		log.Printf("Could not send dm message to %s: %s\n", authorID, err)
		return
	}
}
