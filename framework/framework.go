package framework

import (
	"context"
	admincommands "maquiaBot/handlers/admin-commands"
	"maquiaBot/models"
	osuapi "maquiaBot/osu-api"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Framework struct {
	db  *mongo.Database
	osu *osuapi.Client

	discord  *discordgo.Session // Discord session
	commands map[string]Command // List of commands

	playerCollection *mongo.Collection
	serverCollection *mongo.Collection
	farmCollection   *mongo.Collection

	BeforeAllCommands func(*discordgo.Session, *discordgo.MessageCreate, interface{})
	AfterAllCommands  func(*discordgo.Session, *discordgo.MessageCreate, interface{})
}

func NewFramework(db *mongo.Database, osu *osuapi.Client, discord *discordgo.Session) *Framework {
	framework := &Framework{
		db:       db,
		osu:      osu,
		discord:  discord,
		commands: make(map[string]Command),
	}

	framework.farmCollection = db.Collection("farm")
	framework.playerCollection = db.Collection("players")
	framework.serverCollection = db.Collection("servers")

	// attach handlers
	discord.AddHandler(framework.handleMessageCreate)

	return framework
}

func (f *Framework) RegisterCommand(name string, command Command) {
	f.commands[name] = command
}

func (f *Framework) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	var arbitraryData interface{}

	// stuff that happens before all commands
	if f.BeforeAllCommands != nil {
		f.BeforeAllCommands(s, m, arbitraryData)
	}

	// shit that happens outside of the prefix
	if strings.HasPrefix(m.Content, "maquiaprefix") {
		admincommands.Prefix(s, m)
	} else {
		// get the server prefix
		server, err := f.getServer(m.GuildID)
		if err != nil {
			sentry.CaptureException(err)
			s.ChannelMessageSend(m.ChannelID, "couldn't retrieve server")
			return
		}

		if strings.HasPrefix(m.Content, server.Prefix) {
			restOfMessage := strings.TrimPrefix(m.Content, server.Prefix)

			// now loop through all the commands
			for _, command := range f.commands {
				regex := command.Regex()

				if regex.Match([]byte(restOfMessage)) {
					// set up the context
					ctx := CommandContext{
						S:  s,
						MC: m,

						Players: f.playerCollection,
						Servers: f.serverCollection,
						Farm:    f.farmCollection,

						Osu: f.osu,
						Any: make(map[string]interface{}),
					}

					// call the function
					command.Handle(&ctx)

					break
				}
			}
		}
	}

	// stuff that happens after all commands
	if f.AfterAllCommands != nil {
		f.AfterAllCommands(s, m, arbitraryData)
	}
}

func (f *Framework) getServer(guildID string) (*models.Server, error) {
	var server models.Server
	err := f.serverCollection.
		FindOne(context.TODO(), bson.M{"_id": guildID}).
		Decode(&server)

	// no results
	if err == mongo.ErrNoDocuments {
		server := models.DefaultServerData(guildID)
		// write this to the server
		_, err := f.serverCollection.InsertOne(context.TODO(), server)
		if err != nil {
			return nil, errors.Wrap(err, "failed to insert default server object")
		}
		return server, nil
	}

	// some other error
	if err != nil {
		return nil, errors.Wrap(err, "failed to find server by id")
	}

	// found the right doc
	return &server, nil
}
