package commands

import (
	"context"
	"maquiaBot/framework"
	"maquiaBot/models"
	osuapi "maquiaBot/osu-api"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	usernameRegex = regexp.MustCompile(`(?i)(.+)(link|set)(\s+<@\S+)?(\s+.+)?`)
)

type LinkT struct{}

func Link() LinkT {
	return LinkT{}
}

func (m LinkT) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: link / set"
	embed.Description = "`[osu] (link|set) [@mention] <osu! username>` lets you link an osu! account with the username given to your discord account."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[@mention]",
			Value:  "The person to link the osu! user to **(REQUIRES ADMIN PERMS)**.",
			Inline: true,
		},
		{
			Name:   "<osu! username>",
			Value:  "The username of the osu! player to link to.",
			Inline: true,
		},
	}
}

func (m LinkT) Handle(ctx *framework.CommandContext) int {
	discordUser := ctx.MC.Author
	osuUsername := strings.TrimSpace(usernameRegex.FindStringSubmatch(ctx.MC.Content)[4])

	// farmData := structs.FarmData{}
	// f, err := ioutil.ReadFile("./data/osuData/mapFarm.json")
	// tools.ErrRead(s, err)
	// _ = json.Unmarshal(f, &farmData)

	// Obtain server and check admin permissions for linking with mentions involved
	isAdmin := ctx.Any["isServerAdmin"].(bool)
	if !isAdmin && len(ctx.MC.Mentions) > 0 {
		ctx.Reply("You must be an admin, server manager, or server owner!")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	if len(ctx.MC.Mentions) > 0 {
		discordUser = ctx.MC.Mentions[0]
	}

	// Find the user corresponding to the requesting dicord user
	var player models.Player
	err := ctx.Players.
		FindOne(context.TODO(), bson.M{"discord_id": discordUser.ID}).
		Decode(&player)

	// no players found
	if err == mongo.ErrNoDocuments {
		// Run through the player cache to find the user using the osu! username if no discord ID exists.
		// TODO: don't ignore this case
		// for i, player := range cache {
		// 	if strings.ToLower(player.Osu.Username) == strings.ToLower(osuUsername) && player.Discord.ID == "" {
		// 		player.Discord = *discordUser
		// 		player.FarmCalc(ctx.Osu, ctx.Farm)
		// 		cache[i] = player

		// 		jsonCache, err := json.Marshal(cache)
		// 		tools.ErrRead(s, err)

		// 		err = ioutil.WriteFile("./data/osuData/profileCache.json", jsonCache, 0644)
		// 		tools.ErrRead(s, err)

		// 		if len(ctx.MC.Mentions) >= 1 {
		// 			ctx.Reply("osu! account **%s** has been linked to %s's account!", osuUsername, discordUser.Username)
		// 		} else {
		// 			ctx.Reply("osu! account **%s** has been linked to your discord account!", osuUsername)
		// 		}
		// 		return framework.MIDDLEWARE_RESPONSE_OK
		// 	}
		// }

		// Create player
		user, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: osuUsername,
		})
		if err != nil {
			ctx.Reply("Player: **%s** may not exist!", osuUsername)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		player := models.Player{
			Osu:     *user,
			Discord: *discordUser,
		}

		// Farm calc
		player.FarmCalc(ctx.Osu, ctx.Farm)

		// Save player to mongo
		_, err = ctx.Players.InsertOne(context.TODO(), player)
		if err != nil {
			evt := sentry.CaptureException(err)
			ctx.Reply("couldn't update user in database: %+v", evt)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}

		// cache = append(cache, player)
		// jsonCache, err := json.Marshal(cache)
		// tools.ErrRead(s, err)

		// err = ioutil.WriteFile("./data/osuData/profileCache.json", jsonCache, 0644)
		// tools.ErrRead(s, err)

		if len(ctx.MC.Mentions) >= 1 {
			ctx.Reply("osu! account **%s** has been linked to %s's account!", osuUsername, discordUser.Username)
			return framework.MIDDLEWARE_RESPONSE_OK
		}

		ctx.Reply("osu! account **%s** has been linked to your discord account!", osuUsername)
		return framework.MIDDLEWARE_RESPONSE_OK

	}

	// some other error
	if err != nil {
		ctx.Reply("unexpected error")
		sentry.CaptureException(err)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// player has been found
	if strings.ToLower(player.Osu.Username) == strings.ToLower(osuUsername) {
		if len(ctx.MC.Mentions) >= 1 {
			ctx.Reply("osu! account **%s** already been linked to %s's account!", osuUsername, discordUser.Username)
		} else {
			ctx.Reply("osu! account **%s** already linked to your discord account!", osuUsername)
		}
		return framework.MIDDLEWARE_RESPONSE_OK
	}

	user, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
		Username: osuUsername,
	})
	if err != nil {
		ctx.Reply("Player: **%s** may not exist!", osuUsername)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	player.Osu = *user
	player.FarmCalc(ctx.Osu, ctx.Farm)

	// commit player back to mongo
	_, err = ctx.Players.UpdateOne(context.TODO(), bson.M{"discord_id": discordUser.ID}, &player)
	if err != nil {
		evt := sentry.CaptureException(err)
		ctx.Reply("unexpected error saving player details: %s", evt)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// Remove any accounts of the same user or empty osu! user and with no discord linked
	ctx.Players.DeleteMany(context.TODO(), bson.D{
		{"discord_id", player.Discord.ID},
		{"$or", []interface{}{
			bson.D{{"osu.username", ""}},
			// TODO: get the case-insensitive version back in here
		}},
	})
	// for j := 0; j < len(cache); j++ {
	// 	if player.Discord.ID == "" && (cache[j].Osu.Username == "" || strings.ToLower(cache[j].Osu.Username) == strings.ToLower(osuUsername)) {
	// 		cache = append(cache[:j], cache[j+1:]...)
	// 		j--
	// 	}
	// }

	// jsonCache, err := json.Marshal(cache)
	// tools.ErrRead(s, err)

	// err = ioutil.WriteFile("./data/osuData/profileCache.json", jsonCache, 0644)
	// tools.ErrRead(s, err)

	if len(ctx.MC.Mentions) >= 1 {
		ctx.Reply("osu! account **%s** has been linked to %s's account!", osuUsername, discordUser.Username)
	} else {
		ctx.Reply("osu! account **%s** has been linked to your discord account!", osuUsername)
	}
	return framework.MIDDLEWARE_RESPONSE_OK

}
