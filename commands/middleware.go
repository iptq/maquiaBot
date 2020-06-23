package commands

import (
	"fmt"
	"maquiaBot/framework"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
)

type _InServer struct {
	required bool
}

func InServer(required bool) _InServer {
	return _InServer{required}
}

func (m _InServer) Handle(ctx *framework.CommandContext) int {
	inServer := len(ctx.MC.GuildID) == 0

	if m.required && !inServer {
		ctx.Reply("this command must be used in a server")
		return framework.MIDDLEWARE_RESPONSE_ERR
	} else {
		ctx.Any["inServer"] = inServer
		return framework.MIDDLEWARE_RESPONSE_CONT
	}
}

type _IsServerAdmin struct {
	required bool
}

func IsServerAdmin(required bool) _IsServerAdmin {
	return _IsServerAdmin{required}
}

func (m _IsServerAdmin) Handle(ctx *framework.CommandContext) int {
	isServerAdmin := false

	// get guild info
	guild, err := ctx.S.Guild(ctx.MC.GuildID)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed", err)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	if guild.OwnerID == ctx.MC.Author.ID {
		isServerAdmin = true
	} else {
		// get roles of member
		member, err := ctx.S.GuildMember(ctx.MC.GuildID, ctx.MC.Author.ID)
		if err != nil {
			sentry.CaptureException(err)
			fmt.Println("failed", err)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}

		for _, roleID := range member.Roles {
			role, err := ctx.S.State.Role(ctx.MC.GuildID, roleID)
			if err != nil {
				fmt.Println("failed", err)
				sentry.CaptureException(err)
				continue
			}

			fmt.Println(role.Name, role.Permissions)
			if role.Permissions&(discordgo.PermissionAdministrator|discordgo.PermissionManageServer) > 0 {
				isServerAdmin = true
				break
			}
		}
	}

	if m.required && !isServerAdmin {
		ctx.Reply("you must be server admin to perform this action")
		return framework.MIDDLEWARE_RESPONSE_ERR
	} else {
		ctx.Any["isServerAdmin"] = isServerAdmin
		return framework.MIDDLEWARE_RESPONSE_CONT
	}
}
