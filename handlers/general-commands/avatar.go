package gencommands

import (
	"bytes"
	"image/png"
	"regexp"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// Avatar gets the avatar of the user/referenced user
func Avatar(s *discordgo.Session, m *discordgo.MessageCreate) {
	userRegex, _ := regexp.Compile(`(a|ava|avatar)\s+(.+)`)
	serverRegex, _ := regexp.Compile(`(-s\s|-s$)`)

	users := m.Mentions
	if serverRegex.MatchString(m.Content) {
		ava, err := s.GuildIcon(m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "This is not a server!")
			return
		}

		imgBytes := new(bytes.Buffer)
		_ = png.Encode(imgBytes, ava)
		s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Content: "Here is the server avatar:",
			Files: []*discordgo.File{
				&discordgo.File{
					Name:   "image.png",
					Reader: imgBytes,
				},
			},
		})
	} else if len(users) > 0 {
		var names []string
		var avatars []string
		for _, user := range users {
			names = append(names, user.Username)
			avatars = append(avatars, user.AvatarURL(""))
		}
		postAva(s, m, names, avatars, true)
		return
	} else if userRegex.MatchString(m.Content) {
		username := userRegex.FindStringSubmatch(m.Content)[2]
		discordUser, err := s.User(username)
		if err == nil {
			postAva(s, m, []string{discordUser.Username}, []string{discordUser.AvatarURL("")}, true)
			return
		}

		server, err := s.Guild(m.GuildID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "This is not a server!")
			return
		}

		// Run through usernames, if no match is found, run through member names, if no match is found, send the message author's avatar
		sort.Slice(server.Members, func(i, j int) bool {
			time1, _ := server.Members[i].JoinedAt.Parse()
			time2, _ := server.Members[j].JoinedAt.Parse()
			return time1.Unix() < time2.Unix()
		})
		for _, member := range server.Members {
			if strings.HasPrefix(strings.ToLower(member.User.Username), username) || strings.HasPrefix(strings.ToLower(member.Nick), username) {
				discordUser, _ = s.User(member.User.ID)
				postAva(s, m, []string{member.Nick}, []string{discordUser.AvatarURL("")}, true)
				return
			}
		}
		postAva(s, m, []string{username}, []string{m.Author.AvatarURL("")}, false)
		return
	}
	postAva(s, m, []string{}, []string{m.Author.AvatarURL("")}, true)
}

func postAva(s *discordgo.Session, m *discordgo.MessageCreate, name, avatarURL []string, found bool) {
	negateRegex, _ := regexp.Compile(`-(np|noprev(iew)?)`)
	if len(name) == 0 {
		if negateRegex.MatchString(m.Content) {
			s.ChannelMessageSend(m.ChannelID, "Your avatar is: <"+avatarURL[0]+">")
		} else {
			s.ChannelMessageSend(m.ChannelID, "Your avatar is: "+avatarURL[0])
		}
	} else if len(name) == 1 {
		if found {
			if negateRegex.MatchString(m.Content) {
				s.ChannelMessageSend(m.ChannelID, name[0]+"'s avatar is: <"+avatarURL[0]+">")
			} else {
				s.ChannelMessageSend(m.ChannelID, name[0]+"'s avatar is: "+avatarURL[0])
			}
		} else {
			if negateRegex.MatchString(m.Content) {
				s.ChannelMessageSend(m.ChannelID, "No person named "+name[0]+", Your avatar is: <"+avatarURL[0]+">")
			} else {
				s.ChannelMessageSend(m.ChannelID, "No person named "+name[0]+", Your avatar is: "+avatarURL[0])
			}
		}
	} else {
		message := ""
		if negateRegex.MatchString(m.Content) {
			for i := range name {
				message += name[i] + "'s avatar is: <" + avatarURL[i] + ">\n"
			}
		} else {
			for i := range name {
				message += name[i] + "'s avatar is: " + avatarURL[i] + "\n"
			}
		}
		s.ChannelMessageSend(m.ChannelID, message)
	}
}
