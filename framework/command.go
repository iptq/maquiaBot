package framework

import (
	"context"
	"fmt"
	"maquiaBot/models"
	osuapi "maquiaBot/osu-api"
	"regexp"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Command interface {
	// Returns a regex to match on this command
	Regex() *regexp.Regexp

	// Handle the input
	Handle(*CommandContext) int

	// Help adds help information to a MessageEmbed
	Help(*discordgo.MessageEmbed)
}

type CommandContext struct {
	S  *discordgo.Session
	MC *discordgo.MessageCreate

	Players *mongo.Collection
	Servers *mongo.Collection
	Farm    *mongo.Collection

	Server *models.Server
	Osu    *osuapi.Client
	Any    map[string]interface{}
}

// Shortcut function for sending a reply to the original sender
func (c *CommandContext) Reply(format string, params ...interface{}) (*discordgo.Message, error) {
	message := fmt.Sprintf(format, params...)
	return c.S.ChannelMessageSend(c.MC.ChannelID, message)
}

// Shortcut function for sending a reply to the original sender with an error
func (c *CommandContext) ReplyErr(err error, format string, params ...interface{}) (*discordgo.Message, error) {
	evt := sentry.CaptureException(err)
	message := fmt.Sprintf(format, params...)
	message = fmt.Sprintf("%s (error id: %+v)", message, *evt)
	return c.S.ChannelMessageSend(c.MC.ChannelID, message)
}

// Send the error
func (c *CommandContext) SendErr(err error) {
	sentry.CaptureException(err)
}

// Get osu! information associated of author
func (c *CommandContext) GetOsuProfile() (models.Player, error) {
	var player models.Player
	err := c.Players.FindOne(context.TODO(), bson.M{
		"discord.id": c.MC.Author.ID,
	}).Decode(&player)
	if err != nil {
		return models.Player{}, err
	}
	return player, nil
}
