package commands

import (
	"context"
	"maquiaBot/framework"
	"maquiaBot/models"
	osuapi "maquiaBot/osu-api"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	usernameRegex = regexp.MustCompile(`(?i)([^ls]+)(link|set)(\s+<@\S+)?(\s+.+)?`)
)

type _Link struct{}

func Link() _Link {
	return _Link{}
}

func (m _Link) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: link / set"
	embed.Description = "`(link|set) [@mention] <osu! username>` lets you link an osu! account with the username given to your discord account."
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

func (m _Link) Handle(ctx *framework.CommandContext) int {
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
		FindOne(context.TODO(), bson.M{
			"discord.id": discordUser.ID,
		}).
		Decode(&player)

	// no players found
	if err == mongo.ErrNoDocuments {
		// Create player
		user, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: osuUsername,
		})
		if err != nil {
			ctx.ReplyErr(err, "Player: **%s** may not exist!", osuUsername)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		osuUsername = user.Username
		player := models.Player{
			ID:      primitive.NewObjectID(),
			Osu:     *user,
			Discord: *discordUser,
		}

		// Farm calc
		player.FarmCalc(ctx.Osu, ctx.Farm)

		// Save player to mongo
		_, err = ctx.Players.InsertOne(context.TODO(), player)
		if err != nil {
			ctx.ReplyErr(err, "couldn't update user in database")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}

		if len(ctx.MC.Mentions) >= 1 {
			ctx.Reply("osu! account **%s** has been linked to %s's account!", osuUsername, discordUser.Username)
			return framework.MIDDLEWARE_RESPONSE_OK
		}

		ctx.Reply("osu! account **%s** has been linked to your discord account!", osuUsername)
		return framework.MIDDLEWARE_RESPONSE_OK
	}

	// some other error
	if err != nil {
		ctx.ReplyErr(err, "unexpected error")
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
		ctx.ReplyErr(err, "Player: **%s** may not exist!", osuUsername)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	osuUsername = user.Username
	player.Osu = *user
	player.FarmCalc(ctx.Osu, ctx.Farm)

	// commit player back to mongo
	_, err = ctx.Players.ReplaceOne(
		context.TODO(),
		bson.M{"_id": player.ID},
		&player,
	)
	if err != nil {
		ctx.ReplyErr(err, "unexpected error saving player details")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	if len(ctx.MC.Mentions) >= 1 {
		ctx.Reply("osu! account **%s** has been linked to %s's account!", osuUsername, discordUser.Username)
	} else {
		ctx.Reply("osu! account **%s** has been linked to your discord account!", osuUsername)
	}
	return framework.MIDDLEWARE_RESPONSE_OK

}
