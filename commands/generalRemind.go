package commands

import (
	"encoding/json"
	"io/ioutil"
	"maquiaBot/framework"
	"maquiaBot/structs"
	"maquiaBot/tools"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type _Remind struct{}

func Remind() _Remind {
	return _Remind{}
}

func (m _Remind) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: remind / reminder"
	embed.Description = "`(remind|reminder) [text] [in time]` reminds you in some amount of time."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[text]",
			Value:  "The text to remind you about. Not required.",
			Inline: true,
		},
		{
			Name:   "[in time]",
			Value:  "The time until you want to be reminded.",
			Inline: true,
		},
		{
			Name:   "Example format:",
			Value:  "`$remind play osu! in 5 hours` Will remind you about `play osu!` in 5 hours.",
			Inline: true,
		},
		{
			Name:  "Related Commands:",
			Value: "`reminders`, `remindremove`",
		},
	}
}

func (m _Remind) Handle(ctx *framework.CommandContext) int {
	remindRegex, _ := regexp.Compile(`(?i)remind(er)?\s+(.+)`)
	timeRegex, _ := regexp.Compile(`(?i)\s(\d+) (month|week|day|hour|minute|second)s?`)
	dateRegex, _ := regexp.Compile(`(?i)at\s+(.+)`)
	reminderTime := time.Duration(0)
	text := ""
	timeResultString := ""
	// Parse info
	if remindRegex.MatchString(m.Content) {
		text = remindRegex.FindStringSubmatch(m.Content)[2]
		if timeRegex.MatchString(m.Content) {
			times := timeRegex.FindAllStringSubmatch(m.Content, -1)
			months := 0
			weeks := 0
			days := 0
			hours := 0
			minutes := 0
			seconds := 0
			for _, timeString := range times {
				timeVal, err := strconv.Atoi(timeString[1])
				if err != nil {
					break
				}
				timeUnit := timeString[2]
				switch timeUnit {
				case "month":
					months += timeVal
				case "week":
					weeks += timeVal
				case "day":
					days += timeVal
				case "hour":
					hours += timeVal
				case "minute":
					minutes += timeVal
				case "second":
					seconds += timeVal
				}
				text = strings.Replace(text, strings.TrimSpace(timeString[0]), "", 1)
				text = strings.TrimSpace(text)
				text = strings.TrimSuffix(text, "and")
				text = strings.TrimSuffix(text, ",")
			}
			text = strings.TrimSpace(text)
			text = strings.TrimSuffix(text, "in")
			text = strings.TrimSpace(text)
			reminderTime += time.Second * time.Duration(months) * 2629744
			reminderTime += time.Second * time.Duration(weeks) * 604800
			reminderTime += time.Second * time.Duration(days) * 86400
			reminderTime += time.Second * time.Duration(hours) * 3600
			reminderTime += time.Second * time.Duration(minutes) * 60
			reminderTime += time.Second * time.Duration(seconds)
		} else if dateRegex.MatchString(m.Content) {
			// Parse date
			date := dateRegex.FindStringSubmatch(m.Content)[1]
			t, err := tools.TimeParse(date)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, "Invalid datetime format!")
				return
			}

			reminderTime = t.Sub(time.Now())
			text = dateRegex.ReplaceAllString(text, "")
		}
	}
	if reminderTime == 0 { // Default to 5 minutes
		reminderTime = time.Second * time.Duration(300)
	}
	// Check if time duration is dumb as hell
	if reminderTime.Hours() > 8760 {
		s.ChannelMessageSend(m.ChannelID, "Ur really funny mate")
		return
	}

	// Obtain date
	timeResult := time.Now().UTC().Add(reminderTime)
	timeResultString = timeResult.Format(time.UnixDate)
	text = strings.ReplaceAll(text, "`", "")

	// People can add huge time durations where the time may go backward
	if timeResult.Before(time.Now()) {
		s.ChannelMessageSend(m.ChannelID, "Ur really funny mate")
		return
	}

	// Create reminder and add to list of reminders
	reminder := structs.NewReminder(timeResult, *m.Author, text)
	reminders := []structs.Reminder{}
	_, err := os.Stat("./data/reminders.json")
	if err == nil {
		f, err := ioutil.ReadFile("./data/reminders.json")
		tools.ErrRead(s, err)
		_ = json.Unmarshal(f, &reminders)
	} else {
		s.ChannelMessageSend(m.ChannelID, "An error occurred obtaining reminder data! Please try later.")
		return
	}
	reminders = append(reminders, reminder)
	reminderTimer := structs.ReminderTimer{
		Reminder: reminder,
		Timer:    *time.NewTimer(timeResult.Sub(time.Now().UTC())),
	}
	ReminderTimers = append(ReminderTimers, reminderTimer)

	// Save reminders
	jsonCache, err := json.Marshal(reminders)
	tools.ErrRead(s, err)

	err = ioutil.WriteFile("./data/reminders.json", jsonCache, 0644)
	tools.ErrRead(s, err)

	if text != "" {
		s.ChannelMessageSend(m.ChannelID, "Ok I'll remind you about `"+reminder.Info+"` on "+timeResultString+"\nPlease make sure your DMs are open or else you will not receive the reminder!")
	} else {
		s.ChannelMessageSend(m.ChannelID, "Ok I'll remind you on "+timeResultString+"\nPlease make sure your DMs are open or else you will not receive the reminder!")
	}
	// Run reminder
	go RunReminder(s, reminderTimer)
	return framework.MIDDLEWARE_RESPONSE_OK
}
