package helpcommands

import (
	"github.com/bwmarrin/discordgo"
)

// PPAdd explains the ppadd functionality
func PPAdd(embed *discordgo.MessageEmbed) *discordgo.MessageEmbed {
	embed.Author.Name = "Command: ppadd / addpp"
	embed.Description = "`(ppadd|addpp) [osu! username] <pp amount>` shows how much more pp you would have if you obtained a score with this amount of pp."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[osu! username]",
			Value:  "The osu! user to check. No user given will use the account linked to your discord account.",
			Inline: true,
		},
		{
			Name:   "<pp amount>",
			Value:  "The amount of pp to add to check.",
			Inline: true,
		},
	}
	return embed
}
