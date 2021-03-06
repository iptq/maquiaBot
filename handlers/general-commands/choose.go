package gencommands

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Choose chooses from options for you
func Choose(s *discordgo.Session, m *discordgo.MessageCreate) {
	regex, _ := regexp.Compile(`(?i)(ch(oose)?)\s(.+)`)
	if !regex.MatchString(m.Content) {
		s.ChannelMessageSend(m.ChannelID, "Please give options to choose from!")
		return
	}

	choices := strings.Split(regex.FindStringSubmatch(m.Content)[3], "|")
	for i := range choices {
		choices[i] = strings.TrimSpace(choices[i])
	}
	roll, _ := rand.Int(rand.Reader, big.NewInt(int64(len(choices))))
	s.ChannelMessageSend(m.ChannelID, choices[roll.Int64()])
}
