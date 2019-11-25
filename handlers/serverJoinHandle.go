package handlers

import (
	"github.com/bwmarrin/discordgo"
)

// ServerJoin is to send a message when the bot joins a server
func ServerJoin(s *discordgo.Session, g *discordgo.GuildCreate) {
	channels := g.Channels
	for _, channel := range channels {
		_, err := s.ChannelMessageSend(channel.ID, "Hello! My default prefix is `$` but you can change it by using `$prefix` or `maquiaprefix`\nPlease remember that this bot is still currently under development so this bot may constantly go on and off as more features are being added!")
		if err == nil {
			return
		}
	}
}
