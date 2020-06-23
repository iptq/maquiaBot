package framework

import (
	"fmt"
	osuapi "maquiaBot/osu-api"
	"regexp"

	"github.com/bwmarrin/discordgo"
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

	Osu *osuapi.Client
	Any map[string]interface{}
}

// Shortcut function for sending a reply to the original sender
func (c *CommandContext) Reply(format string, params ...interface{}) (*discordgo.Message, error) {
	message := fmt.Sprintf(format, params...)
	return c.S.ChannelMessageSend(c.MC.ChannelID, message)
}
