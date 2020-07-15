package commands

import (
	"maquiaBot/framework"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	userRegex = regexp.MustCompile(`(?i)info\s+(.+)`)
)

type _Info struct{}

func Info() _Info {
	return _Info{}
}

func (m _Info) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: info"
	embed.Description = "`info [username]` gets the information for a user."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:  "[username]",
			Value: "Gets the info for the given username / nickname / ID. Gives your info if no username / nickname / ID is given",
		},
		{
			Name:  "Related Commands:",
			Value: "`roleinfo`, `serverinfo`",
		},
	}
}

func (m _Info) Handle(ctx *framework.CommandContext) int {
	userTest := ""
	user := ctx.MC.Author
	nickname := "N/A"
	roles := "N/A"
	var joinDate discordgo.Timestamp
	var err error
	if len(ctx.MC.Mentions) == 1 {
		user = ctx.MC.Mentions[0]
	} else {
		if userRegex.MatchString(ctx.MC.Content) {
			userTest = userRegex.FindStringSubmatch(ctx.MC.Content)[1]
		}
		user, err = ctx.S.User(userTest)
		if err == nil {
			userTest = user.Username
		} else {
			user = ctx.MC.Author
		}
	}

	members, err := ctx.S.GuildMembers(ctx.MC.GuildID, "", 1000)
	if err == nil {
		if userTest == "" {
			for _, member := range members {
				if member.User.ID == ctx.MC.Author.ID {
					user, _ = ctx.S.User(member.User.ID)
					nickname = member.Nick
					joinDate = member.JoinedAt
					roles = ""
					for _, role := range member.Roles {
						discordRole, err := ctx.S.State.Role(ctx.MC.GuildID, role)
						if err != nil {
							continue
						}
						roles = roles + discordRole.Name + ", "
					}
					if roles != "" {
						roles = roles[:len(roles)-2]
					}
				}
			}
		} else {
			sort.Slice(members, func(i, j int) bool {
				time1, _ := members[i].JoinedAt.Parse()
				time2, _ := members[j].JoinedAt.Parse()
				return time1.Unix() < time2.Unix()
			})
			for _, member := range members {
				if strings.HasPrefix(strings.ToLower(member.User.Username), strings.ToLower(userTest)) || strings.HasPrefix(strings.ToLower(member.Nick), strings.ToLower(userTest)) {
					user, _ = ctx.S.User(member.User.ID)
					nickname = member.Nick
					joinDate = member.JoinedAt
					roles = ""
					for _, role := range member.Roles {
						discordRole, err := ctx.S.State.Role(ctx.MC.GuildID, role)
						if err != nil {
							continue
						}
						roles = roles + discordRole.Name + ", "
					}
					if roles != "" {
						roles = roles[:len(roles)-2]
					}
					break
				}
			}
		}
	}

	// Reformat joinDate
	joinDateDate, _ := joinDate.Parse()
	serverCreateDate, _ := discordgo.SnowflakeTimestamp(ctx.MC.GuildID)
	joinDateString := "N/A"
	if joinDateDate.After(serverCreateDate) {
		joinDateString = joinDateDate.Format(time.RFC822Z)
	}

	// Created at date
	createdAt, _ := discordgo.SnowflakeTimestamp(user.ID)

	// Status
	presence, err := ctx.S.State.Presence(ctx.MC.GuildID, user.ID)
	status := "Offline"
	if err == nil {
		status = strings.Title(string(presence.Status))
	}

	// Obtain osu! info
	osuUsername := "N/A"
	player, err := ctx.GetOsuProfile()
	if err != nil && err != mongo.ErrNoDocuments {
		ctx.ReplyErr(err, "couldn't retrieve discord user info")
	} else {
		osuUsername = player.Osu.Username
	}

	// Fix any blanks
	if roles == "" {
		roles = "None"
	}
	if nickname == "" {
		nickname = "None"
	}

	ctx.S.ChannelMessageSendEmbed(ctx.MC.ChannelID, &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    user.String(),
			IconURL: user.AvatarURL("2048"),
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL("2048"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "ID",
				Value:  user.ID,
				Inline: true,
			},
			{
				Name:   "Nickname",
				Value:  nickname,
				Inline: true,
			},
			{
				Name:   "Account Created",
				Value:  createdAt.UTC().Format(time.RFC822Z),
				Inline: true,
			},
			{
				Name:   "Date Joined",
				Value:  joinDateString,
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  status,
				Inline: true,
			},
			{
				Name:   "Linked on osu! as",
				Value:  osuUsername,
				Inline: true,
			},
			{
				Name:  "Roles",
				Value: roles,
			},
		},
	})

	return framework.MIDDLEWARE_RESPONSE_OK
}
