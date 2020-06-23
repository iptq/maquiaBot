package commands

import (
	"maquiaBot/framework"

	"github.com/bwmarrin/discordgo"
)

type _Profile struct {
}

func Profile() _Profile {
	return _Profile{}
}

func (m _Profile) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: osu / profile"
	embed.Description = "`(osu|[osu] profile|<profile link>) [osu! username] [-m <mode>]` lets you obtain user information."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "<profile link>",
			Value:  "You may link a map instead of using `osu` or `profile` to get user information.",
			Inline: true,
		},
		{
			Name:   "[osu! username]",
			Value:  "The username to look for. Using a link will use the user linked instead. No user linked for `osu` or `profile` messages will use the user linked to your discord account.",
			Inline: true,
		},
		{
			Name:   "[-m <mode>]",
			Value:  "The mode to show user information for (Default: osu!standard).",
			Inline: true,
		},
		{
			Name:  "Related Commands:",
			Value: "`osudetail`, `osutop`",
		},
	}
}

func (m _Profile) Handle(ctx *framework.CommandContext) int {
	ctx.Reply("lol not implemented yet")
	return framework.MIDDLEWARE_RESPONSE_OK
}
