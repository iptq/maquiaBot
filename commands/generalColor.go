package commands

import (
	"log"
	colourtools "maquiaBot/colour-tools"
	"maquiaBot/framework"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	colorRegex = regexp.MustCompile(`(?i)(color|colour)\s+(.+)`)
	colorList  = "https://www.w3schools.com/colors/colors_names.asp"
)

type _Color struct{}

func Color() _Color {
	return _Color{}
}

func (m _Color) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: color / colour"
	embed.Description = "`(color|colour) <color>` sets your color."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:  "<color>",
			Value: "The color you want to change to. The available color names are listed on this page: " + colorList + " (ex: !color DodgerBlue)",
		},
	}
}

func (m _Color) Handle(ctx *framework.CommandContext) int {
	var colorName string

	// parse color name
	if colorRegex.MatchString(ctx.MC.Content) {
		colorName = colorRegex.FindStringSubmatch(ctx.MC.Content)[2]
		colorName = strings.ToLower(colorName)
	} else {
		ctx.Reply("you need to choose a color. pick one here: %s", colorList)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// validate color name
	colorValue, ok := colourtools.Colors[colorName]
	if !ok {
		ctx.Reply("that's not in the list of available colors: %s", colorList)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// get roles
	channel, err := ctx.S.Channel(ctx.MC.ChannelID)
	if err != nil {
		ctx.ReplyErr(err, "couldn't get channel info")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	existingRoles, err := ctx.S.GuildRoles(channel.GuildID)
	if err != nil {
		ctx.ReplyErr(err, "couldn't get existing roles")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	var colorRole *discordgo.Role
	var colorRoleFound = false
	roleMap := make(map[string]string)
	for _, role := range existingRoles {
		roleMap[role.ID] = role.Name
		if role.Name == "Color: "+colorName {
			colorRole = role
			colorRoleFound = true
		}
	}

	if !colorRoleFound {
		// create the role
		newRole, err := ctx.S.GuildRoleCreate(channel.GuildID)
		if err != nil {
			ctx.ReplyErr(err, "couldn't create role")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		role, err := ctx.S.GuildRoleEdit(
			channel.GuildID,     // guild id
			newRole.ID,          // role id
			"Color: "+colorName, // name of the role
			colorValue,          // color
			false,               // whether to display separately
			0,                   // permissions
			false,               // mentionable
		)
		colorRole = role
		if err != nil {
			ctx.ReplyErr(err, "couldn't edit role")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	}

	member, err := ctx.S.GuildMember(channel.GuildID, ctx.MC.Author.ID)
	if err != nil {
		ctx.ReplyErr(err, "couldn't get member info")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// remove existing colors
	for _, roleID := range member.Roles {
		role, ok := roleMap[roleID]
		log.Println(ok, roleID)
		if ok && strings.HasPrefix(role, "Color: ") {
			err := ctx.S.GuildMemberRoleRemove(channel.GuildID, ctx.MC.Author.ID, roleID)
			if err != nil {
				ctx.ReplyErr(err, "couldn't remove role %s from user", roleID)
				return framework.MIDDLEWARE_RESPONSE_ERR
			}
		}
	}

	// add current role
	err = ctx.S.GuildMemberRoleAdd(channel.GuildID, ctx.MC.Author.ID, colorRole.ID)
	if err != nil {
		ctx.ReplyErr(err, "couldn't set role to user")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	ctx.Reply("nice! now you are **%s**", colorName)
	return framework.MIDDLEWARE_RESPONSE_OK
}
