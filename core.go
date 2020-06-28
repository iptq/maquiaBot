package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"syscall"
	"time"

	c "maquiaBot/commands"
	config "maquiaBot/config"
	"maquiaBot/framework"
	gencommands "maquiaBot/handlers/general-commands"
	osucommands "maquiaBot/handlers/osu-commands"
	"maquiaBot/logging"
	osuapi "maquiaBot/osu-api"
	osutools "maquiaBot/osu-tools"
	structs "maquiaBot/structs"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func main() {
	// Open the config
	config.NewConfig("config/config.json")

	// Set up sentry logging
	if len(config.Conf.SentryDSN) > 0 {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:   config.Conf.SentryDSN,
			Debug: true,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to initialize sentry: %+v\n", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	// Open connection to the database
	log.Println("connecting to", config.Conf.MongoHost)
	client, err := mongo.NewClient(options.Client().ApplyURI(config.Conf.MongoHost))
	if err != nil {
		logging.Fatal(err, "couldn't create mongodb client")
	}
	err = client.Connect(context.TODO())
	if err != nil {
		logging.Fatal(err, "couldn't connect to mongodb")
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		// Can't connect to Mongo server
		logging.Fatal(err, "couldn't ping mongodb server")
	}
	db := client.Database(config.Conf.MongoDb)
	defer db.Client().Disconnect(context.TODO())
	log.Println("connected to mongodb")

	// Create data folders and stuff
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		err = os.MkdirAll("./data", 0755)
		readErr(err)
		log.Println("Created data directory.")

		err = ioutil.WriteFile("./data/genitalRecords.json", []byte{}, 0644)
		readErr(err)
		log.Println("Created data/genitalRecords.json.")
		err = ioutil.WriteFile("./data/reminders.json", []byte{}, 0644)
		readErr(err)
		log.Println("Created data/reminders.json.")

		err = os.MkdirAll("./data/channelData", 0755)
		readErr(err)
		log.Println("Created data/channelData directory.")
		err = os.MkdirAll("./data/channelData", 0755)
		readErr(err)
		log.Println("Created data/channelData directory.")

		err = os.MkdirAll("./data/serverData", 0755)
		readErr(err)
		log.Println("Created data/serverData directory.")

		err = os.MkdirAll("./data/osuData", 0755)
		readErr(err)
		log.Println("Created data/osuData directory.")
		err = ioutil.WriteFile("./data/osuData/mapFarm.json", []byte{}, 0644)
		readErr(err)
		log.Println("Created data/osuData/mapFarm.json.")
		err = ioutil.WriteFile("./data/osuData/mapperData.json", []byte{}, 0644)
		readErr(err)
		log.Println("Created data/osuData/mapperData.json.")
		err = ioutil.WriteFile("./data/osuData/profileCache.json", []byte{}, 0644)
		readErr(err)
		log.Println("Created data/osuData/profileCache.json.")
	}

	// Obtain config
	osuAPI := osuapi.NewClient(config.Conf.OsuToken)
	osucommands.OsuAPI = osuAPI
	osutools.OsuAPI = osuAPI
	discord, err := discordgo.New("Bot " + config.Conf.DiscordToken)
	readErr(err)

	// Handle farm data
	go osutools.FarmUpdate(discord)

	// fuck golang
	wrap := func(m framework.Middleware, r string) framework.Command {
		return framework.Wrap(m, r)
	}
	chain := func(base framework.Middleware, ms ...framework.Middleware) framework.Middleware {
		for _, m := range ms {
			base = framework.Chain(base, m)
		}
		return base
	}

	// Initialize the framework for handling events
	f := framework.NewFramework(&config.Conf, db, osuAPI, discord)

	// general commands
	f.RegisterCommand("color", wrap(c.Color(), "^(color|colour)"))
	f.RegisterCommand("info", wrap(chain(c.IsServerAdmin(false), c.Info()), "^info"))

	// osu commands
	f.RegisterCommand("compare", wrap(c.Compare(), "^(c[^o]|compare)"))
	f.RegisterCommand("profile", wrap(c.Profile(), "^(osu|profile)"))
	f.RegisterCommand("recentBest", wrap(chain(c.RecentBest()), "^(rb|recentb|recentbest)"))
	f.RegisterCommand("recent", wrap(chain(c.Recent()), "^(r[^b]|rs|recent)"))
	f.RegisterCommand("set", wrap(chain(c.IsServerAdmin(false), c.Link()), "^(link|set)"))

	// Add handlers
	// discord.AddHandler(handlers.MessageHandler)
	// discord.AddHandler(handlers.ReactAdd)
	// discord.AddHandler(handlers.ServerJoin)
	// discord.AddHandler(handlers.ServerLeave)

	// Open a websocket connection to Discord and begin listening
	for {
		err = discord.Open()
		if err == nil {
			break
		}
	}
	log.Println("Bot is now running in " + strconv.Itoa(len(discord.State.Guilds)) + " servers.")
	discord.UpdateStatus(0, strconv.Itoa(len(discord.State.Guilds))+" servers")

	// Resume all reminder timers
	reminders := []structs.Reminder{}
	_, err = os.Stat("./data/reminders.json")
	if err == nil {
		f, err := ioutil.ReadFile("./data/reminders.json")
		readErr(err)
		_ = json.Unmarshal(f, &reminders)
	} else {
		readErr(err)
	}
	reminderTimers := []structs.ReminderTimer{}
	for _, reminder := range reminders {
		reminderTimer := structs.ReminderTimer{
			Reminder: reminder,
			Timer:    *time.NewTimer(reminder.Target.Sub(time.Now().UTC())),
		}
		reminderTimers = append(reminderTimers, reminderTimer)
		go gencommands.RunReminder(discord, reminderTimer)
	}
	gencommands.ReminderTimers = reminderTimers

	// Get osu! tracking data for channels
	var channels []string

	err = filepath.Walk("./data/channelData", func(path string, info os.FileInfo, err error) error {
		readErr(err)
		channels = append(channels, path)
		return nil
	})
	readErr(err)
	IDregex, _ := regexp.Compile(`(?i)(\d+)\.json`)
	for _, channel := range channels {
		if IDregex.MatchString(channel) {
			chID := IDregex.FindStringSubmatch(channel)[1]
			ch, err := discord.Channel(chID)
			if err == nil {
				go osutools.TrackPost(*ch, discord)
			}
		}
	}

	// Get osu! mapper tracking data
	// go osutools.TrackMapperPost(discord) Commented until a solution is found for its issues

	// Open DB
	// tools.DB, err = gorm.Open("mysql", config.Conf.Database.Username+":"+config.Conf.Database.Password+"@/"+config.Conf.Database.Name)
	// readErr(err)

	// Create a channel to keep the bot running until a prompt is given to close
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Kill)
	<-sc

	// Close sessions
	discord.Close()
	// tools.DB.Close()
}

func readErr(err error) {
	if err != nil {
		pc, fn, line, _ := runtime.Caller(1)
		log.Fatalf("[error] in %s[%s:%d] %v\n", runtime.FuncForPC(pc).Name(), fn, line, err)
	}
}
