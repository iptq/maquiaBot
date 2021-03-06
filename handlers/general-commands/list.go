package gencommands

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// List randomizes a list of objects
func List(s *discordgo.Session, m *discordgo.MessageCreate) {
	list := strings.Split(m.Content, "\n")[1:]

	// Use txt file if given
	if len(m.Attachments) > 0 {
		res, err := http.Get(m.Attachments[0].URL)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to get file information!")
			return
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Unable to parse file information!")
			return
		}

		list = strings.Split(string(b), "\n")
		if len(list) <= 1 {
			s.ChannelMessageSend(m.ChannelID, "Please give a list of lines to randomize! Could not find 2+ lines to randomize from file!")
			return
		}

		if strings.Contains(list[0], "list") {
			list = list[1:]
		}
	} else if len(list) <= 1 {
		s.ChannelMessageSend(m.ChannelID, "Please give a list of lines to randomize!")
		return
	}

	rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })

	_, err := s.ChannelMessageSend(m.ChannelID, strings.Join(list, "\n"))
	if err != nil {
		for i := 0; i < len(list); i++ {
			if len(strings.Join(list[:i], "\n")) > 2000 {
				s.ChannelMessageSend(m.ChannelID, strings.Join(list[:i-1], "\n"))
				list = list[i-1:]
				i = 0
			}
		}
	}
}
