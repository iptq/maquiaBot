package framework

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"maquiaBot/config"
	admincommands "maquiaBot/handlers/admin-commands"
	"maquiaBot/models"
	osuapi "maquiaBot/osu-api"
)

type Framework struct {
	config *config.Config
	db     *mongo.Database
	osu    *osuapi.Client

	discord  *discordgo.Session // Discord session
	commands map[string]Command // List of commands

	playerCollection *mongo.Collection
	serverCollection *mongo.Collection
	farmCollection   *mongo.Collection

	BeforeAllCommands func(*discordgo.Session, *discordgo.MessageCreate, interface{})
	AfterAllCommands  func(*discordgo.Session, *discordgo.MessageCreate, interface{})
}

func NewFramework(config *config.Config, db *mongo.Database, osu *osuapi.Client, discord *discordgo.Session) *Framework {
	framework := &Framework{
		config:   config,
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
			evt := sentry.CaptureException(err)
			s.ChannelMessageSend(m.ChannelID, "couldn't retrieve server: "+string(*evt))
			return
		}

		if strings.HasPrefix(m.Content, server.Prefix) {
			restOfMessage := strings.TrimPrefix(m.Content, server.Prefix)

			// are we asking for help?
			help := false
			if strings.HasPrefix(restOfMessage, "help") {
				restOfMessage = strings.TrimPrefix(restOfMessage, "help")
				restOfMessage = strings.TrimLeft(restOfMessage, " \t")
				help = true
			}

			// now loop through all the commands
			found := false
			embed := f.defaultHelpEmbed(s)
			for _, command := range f.commands {
				regex := command.Regex()

				if regex.Match([]byte(restOfMessage)) {
					found = true

					// are we looking for help?
					if help {
						command.Help(embed)
						fmt.Println("embed", embed)

						// TODO: handle this
						s.ChannelMessageSendEmbed(m.ChannelID, embed)
						break
					}

					// set up the context
					ctx := CommandContext{
						S:  s,
						MC: m,

						Players: f.playerCollection,
						Servers: f.serverCollection,
						Farm:    f.farmCollection,

						Server: server,
						Osu:    f.osu,
						Any:    make(map[string]interface{}),
					}

					// call the function
					command.Handle(&ctx)

					break
				}
			}

			if !found {
				// print generic help
				s.ChannelMessageSendEmbed(m.ChannelID, embed)
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
		FindOne(context.TODO(), bson.M{"server_id": guildID}).
		Decode(&server)

	// no results
	if err == mongo.ErrNoDocuments {
		fmt.Println("no documents found")
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

func (f *Framework) defaultHelpEmbed(s *discordgo.Session) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://discordapp.com/oauth2/authorize?&client_id=" + s.State.User.ID + "&scope=bot&permissions=0",
			Name:    "Click here to invite " + f.config.BotName + "!",
			IconURL: s.State.User.AvatarURL("2048"),
		},
	}
}
